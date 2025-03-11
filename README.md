# MCP Generator

`ai-create-mcp` is a Go-based tool that converts OpenAPI Specification (OAS) files into a Model Context Protocol (MCP) program. It provides a simple and efficient way to generate MCP-compliant code or configurations from OAS files, with customizable options for project setup and integrations.

**[中文](README-zh.md)**

## Features

- Convert OAS files to MCP protocol.
- Customizable project name, directory, and version.
- Optional integration with Claude.app.
- Inspector tool for debugging and analysis.

## Installation

To install the `ai-create-mcp`, ensure you have Go installed on your system. Then, run the following command:

```bash
go install github.com/xxlv/ai-create-mcp
```

Alternatively, clone the repository and build it manually:

```bash
git clone https://github.com/xxlv/ai-create-mcp.git
cd ai-create-mcp
go build
```

## Usage

The tool is configured via command-line flags. Below is the list of available flags and their descriptions:

| Flag           | Type   | Default Value  | Description                           |
| -------------- | ------ | -------------- | ------------------------------------- |
| `-path`        | string | `""`           | Directory to create the project in    |
| `-name`        | string | `""`           | Project name                          |
| `-oaspath`     | string | `""`           | Path to the OAS file                  |
| `-version`     | string | `"0.1.0"`      | Server version                        |
| `-inspector`   | bool   | `true`         | Enable/disable the inspector tool     |
| `-description` | string | `"Simple mcp"` | Project description                   |
| `-claudeapp`   | bool   | `true`         | Enable/disable Claude.app integration |
| `-autoyes`     | bool   | `true`         | Enable/disable auto-confirmation      |

### Example

To generate an MCP project from an OAS file, run:

```bash
ai-create-mcp -path ./myproject -name my-mcp-app -oaspath ./openapi.yaml -version 1.0.0
```

This command will:

- Create a project in the `./myproject` directory.
- Name the project `my-mcp-app`.
- Use the OAS file located at `./openapi.yaml`.
- Set the server version to `1.0.0`.

To disable the inspector and Claude.app integration, use:

```bash
ai-create-mcp -path ./myproject -name my-mcp-app -oaspath ./openapi.yaml -inspector=false -claudeapp=false
```

For a fully automated run (no prompts), enable `-autoyes`:

```bash
ai-create-mcp -path ./myproject -name my-mcp-app -oaspath ./openapi.yaml -autoyes
```

## Configuration

The tool relies on the provided command-line flags for configuration. Ensure that:

- The `-oaspath` points to a valid OAS file (e.g., `.yaml`).
- The `-path` directory is writable.
- The `-name` adheres to valid naming conventions for your use case.

## Contributing

Contributions are welcome! To contribute:

1. Fork the repository.
2. Create a new branch (`git checkout -b feature/your-feature`).
3. Make your changes and commit them (`git commit -m "Add your feature"`).
4. Push to the branch (`git push origin feature/your-feature`).
5. Open a pull request.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## Contact

For questions or support, please open an issue on the GitHub repository.
