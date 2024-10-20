# Wget

This project is a recreation of some functionalities of wget using Go. It allows you to download files from the web, mirror websites, and perform various other download-related tasks.

## Features

- Download a single file from a given URL
- Save downloaded files under different names
- Save files to specific directories
- Set download speed limits
- Download files in the background
- Download multiple files concurrently from a list
- Mirror entire websites

## Usage

### Basic Usage

To download a file:

```
go run . https://example.com/file.zip
```

### Flags

- `-B`: Download in background
  ```
  go run . -B https://example.com/file.zip
  ```

- `-O`: Save file under a different name
  ```
  go run . -O=newname.zip https://example.com/file.zip
  ```

- `-P`: Specify save directory
  ```
  go run . -P=~/Downloads/ https://example.com/file.zip
  ```

- `--rate-limit`: Limit download speed
  ```
  go run . --rate-limit=400k https://example.com/file.zip
  ```

- `-i`: Download multiple files from a list
  ```
  go run . -i=download.txt
  ```

- `--mirror`: Mirror a website
  ```
  go run . --mirror https://example.com
  ```

### Website Mirroring Options

- `-R` or `--reject`: Reject specific file types
  ```
  go run . --mirror -R=jpg,gif https://example.com
  ```

- `-X` or `--exclude`: Exclude specific directories
  ```
  go run . --mirror -X=/assets,/css https://example.com
  ```

- `--convert-links`: Convert links for offline viewing
  ```
  go run . --mirror --convert-links https://example.com
  ```

## Output

The program provides feedback on the download process, including:

- Start time
- Request status
- Content size
- Save location
- Progress bar
- Finish time

Example output:

```
start at 2017-10-14 03:46:06
sending request, awaiting response... status 200 OK
content size: 56370 [~0.06MB]
saving file to: ./file.jpg
 55.05 KiB / 55.05 KiB [==========================] 100.00% 1.24 MiB/s 0s
Downloaded [https://example.com/file.jpg]
finished at 2017-10-14 03:46:07
```

## Building

To build the project:

```
go build
```

## Running Tests

To run the tests:

```
go test ./...
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

[Specify your license here]