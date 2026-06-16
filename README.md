# FTP Cracker - Brute Force Tool

A high-performance FTP credential brute force tool written in Go, designed for authorized security testing and penetration testing purposes.

## Overview

FTP Cracker is a multi-threaded brute force utility that attempts to compromise FTP server credentials using a provided wordlist. The tool features concurrent password testing, real-time progress monitoring, and graceful error handling.

## Features

- **Multi-threaded Architecture**: Configurable concurrent worker threads for faster password testing (default: 30 threads)
- **Real-time Progress Monitoring**: Live progress bar showing attempt count and error rate
- **Customizable Configuration**: Support for various FTP server ports and connection timeouts
- **Verbose Mode**: Detailed logging of connection errors during the brute force process
- **Graceful Signal Handling**: Properly handles interruption signals (Ctrl+C) for clean shutdown
- **Error Handling**: Robust error management with detailed validation and feedback
- **Password List Support**: Supports large password lists with comment line filtering
- **Connection Timeout Control**: Adjustable timeout settings for network operations

## Version

Version: 1.0.0

Authors: GhostGTT666 - Gagaltotal666

## System Requirements

- Go 1.26.4 or higher
- Access to target FTP server (authorized testing only)
- Sufficient system resources for concurrent connections

## Installation

### Build from Source

1. Clone the repository:
```bash
git clone https://github.com/gagaltotal/ftp-cracker-tot.git
cd ftp-cracker-tot
```
2. Install Package Library Go:
```bash
go mod init ftp-cracker-tot
go mod tidy
OR
go get github.com/fatih/color
go get github.com/jlaffaye/ftp
```

3. Build the executable:
```bash
go build -o ftpcracker ftp_cracker.go
```

4. The compiled binary `ftpcracker` will be created in the current directory.

## Usage

![Screen Capture](https://raw.githubusercontent.com/gagaltotal/ftp-cracker-tot/refs/heads/main/Screenshot%20from%202026-06-16%2017-33-15.png)

### Basic Command Structure

```bash
./ftpcracker -host <target> -user <username> -passlist <wordlist> [options]
```

### Command-line Options

```
-host, -H          string       Target host or IP address (required)
-user, -u          string       Username for FTP login (required)
-passlist, -p      string       Path to password wordlist file (required)
-port, -P          int          FTP server port (default: 21)
-threads, -t       int          Number of concurrent worker threads (default: 30, max: 1000)
-timeout           int          Connection timeout in seconds (default: 5)
-verbose, -v       boolean      Enable verbose output for detailed logging
-help, -h          boolean      Display help information
```

### Usage Examples

![Screen Capture](https://raw.githubusercontent.com/gagaltotal/ftp-cracker-tot/refs/heads/main/Screenshot%20from%202026-06-16%2018-57-57.png)

#### Basic Brute Force Attack

```bash
./ftpcracker -host 192.168.1.1 -user admin -passlist wordlist.txt
```

#### Using Short Flags

```bash
./ftpcracker -H 192.168.1.1 -u root -p rockyou.txt
```

#### Optimized for Speed with Custom Threads

```bash
./ftpcracker -host example.com -user testuser -passlist passwords.txt -threads 50
```

#### Custom FTP Port with Extended Timeout

```bash
./ftpcracker -host 192.168.1.100 -user admin -passlist wordlist.txt -port 2121 -timeout 10
```

#### Verbose Mode for Debugging

```bash
./ftpcracker -host 192.168.1.1 -u admin -p wordlist.txt -verbose
```

#### Combined Advanced Options

```bash
./ftpcracker -host ftp.example.com -user admin -passlist rockyou.txt -threads 60 -port 2121 -timeout 8 -verbose
```

## Output

When the brute force attack is running, the tool displays:

- Real-time progress bar showing completion percentage
- Number of password attempts tried
- Total passwords in wordlist
- Failed connection attempts count
- Elapsed time for the operation

### Success Output Example

```
[+] Cracking successfully completed
[+] Found valid credentials:
    Host: 192.168.1.113
    User: admin
    Password: SecurePass123
    Attempts: 1547
    Time: 2m 34s
```

### Failure Output Example

```
[-] Credentials not found
    Attempts: 5000
    Failed Connections: 12
    Time: 5m 23s
```

## Password List Format

The password list should contain one password per line. Empty lines and lines starting with '#' (comments) are automatically ignored.

Example wordlist.txt:
```
# Common passwords
password
123456
admin123
letmein
welcome
# More passwords
qwerty
abc123
```

## Configuration Details

### Thread Configuration

- **Minimum threads**: 1
- **Maximum threads**: 1000
- **Default threads**: 30
- **Recommendation**: Start with 30-50 threads; increase for faster attacks on stable connections

### Timeout Settings

- **Minimum timeout**: 1 second
- **Default timeout**: 5 seconds
- **Recommendation**: Increase for unreliable network conditions or remote servers

### Port Configuration

- **Valid port range**: 1-65535
- **Default FTP port**: 21
- **Alternative FTP ports**: 2121, 8021, 8080

## Dependencies

The tool uses the following Go packages:

- `github.com/fatih/color`: Terminal color output
- `github.com/jlaffaye/ftp`: FTP client library

All dependencies are managed via Go modules (go.mod, go.sum).

## Security Considerations

- **Authorization**: Only use this tool on systems you own or have explicit written permission to test
- **Network Traffic**: FTP credentials are transmitted without encryption; consider using SFTP for sensitive operations
- **Detection**: FTP server logs may record brute force attempts; be aware of security monitoring
- **Rate Limiting**: Some FTP servers implement rate limiting; adjust thread count and timeout accordingly
- **Legal Compliance**: Ensure usage complies with applicable laws and regulations

## Error Handling

The tool provides comprehensive error management:

- Validates all required parameters before execution
- Reports detailed error messages for configuration issues
- Handles network timeouts gracefully
- Manages connection failures without interrupting the attack
- Logs verbose error information in verbose mode

## Performance Notes

- Attack speed depends on network latency, FTP server responsiveness, and thread count
- Increasing thread count improves speed but requires more system resources
- Connection timeout affects overall attack duration; lower values may miss slow servers
- Large password lists should be optimized to prioritize likely credentials

## Troubleshooting

### Connection Refused
- Verify target host and port are correct
- Ensure FTP server is running and accessible
- Check firewall rules blocking FTP connections

### Timeout Errors
- Increase timeout value using `-timeout` flag
- Check network connectivity to target
- Verify target FTP server is responding

### Memory Issues
- Reduce thread count if experiencing high memory usage
- Use smaller password lists or split large lists

## License

This project is provided as-is for authorized security testing purposes only.

## Disclaimer

Unauthorized access to computer systems is illegal. This tool is intended for use only by authorized personnel on systems they own or have explicit written permission to test. The authors assume no liability for misuse or damage caused by this tool.