# anon-traffic-gen

A simple application to generate HTTP and WebSocket traffic.

## Description

This application is designed to generate HTTP and WebSocket traffic to a set of configurable URLs, using a variety of user agents. It can be used for various purposes, such as testing, load generation, or simulating user activity.

The application uses the Cobra library for the command-line interface, Viper for configuration management, and Gorilla WebSocket for WebSocket communication. The main traffic generation logic is handled in the `sendRequests()` function, which alternates between sending HTTP requests and generating WebSocket traffic, with a configurable rate limit.

## Architecture

1. **Configuration Management**: The application uses Viper to load and parse the configuration file, which specifies the URLs, user agents, and rate limit.

2. **Command-Line Interface**: Cobra is used to implement the command-line interface, allowing users to easily run the application with custom configuration options.

3. **Traffic Generation**: The `sendRequests()` function is the core of the application, managing the main traffic generation loop. It alternates between sending HTTP requests using the `sendHTTPRequests()` function and generating WebSocket traffic with the `generateWebSocketTraffic()` function, with a delay for rate limiting.

## Code Structure

1. `Config` struct: Defines the structure of the configuration.
2. `rootCmd`: The main Cobra command that runs the application.
3. `initConfig()` and `loadConfig()`: Functions responsible for loading and parsing the configuration file.
4. `sendRequests()`: The main traffic generation loop.
   - `sendHTTPRequests()`: Sends HTTP GET requests to the configured URLs using the configured user agents.
   - `generateWebSocketTraffic()`: Establishes WebSocket connections to the configured URLs, sends a random message, and immediately closes the connection.
5. `main()`: The entry point of the application, which executes the `rootCmd`.

## Dependencies

The application relies on the following external libraries:

- [Cobra](https://github.com/spf13/cobra): For the command-line interface.
- [Viper](https://github.com/spf13/viper): For configuration management.
- [Gorilla WebSocket](https://github.com/gorilla/websocket): For WebSocket communication.

These dependencies are managed using Go modules.

## Configuration

The application uses a YAML configuration file, which can be specified using the `--config` flag. The default configuration file is `config.yaml`.

The configuration file should have the following structure:

```yaml
urls:
  - "https://example.com"
  - "https://another-example.com"
user_agents:
  - "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.3"
  - "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/603.3.8 (KHTML, like Gecko) Version/10.1.2 Safari/603.3.8"
rate_limit: 1 # Requests per second
```

## Usage

To run the application, use the following command:

```
main --config=config.yaml
```

This will start the traffic generation and continue until you press Enter to stop.

## Contributing

If you find any issues or have suggestions for improvements, feel free to open an issue or submit a pull request on the [anon-traffic-gen](https://github.com/YarBurArt/anon-traffic-gen/tree/main).

## License

This project is licensed under the [MIT License](https://github.com/YarBurArt/anon-traffic-gen/blob/main/LICENSE). Free use for peaceful purposes only.
