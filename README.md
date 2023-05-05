# ChatGPT CLI

This project showcases an implementation of ChatGPT clients with streaming support in a Command-Line Interface (CLI)
environment, demonstrating its practicality and effectiveness.

![a screenshot](resources/vhs.gif)

## Features

* **Interactive streaming mode**: Real-time interaction with the GPT model.
* **Query mode**: Single input-output interactions with the GPT model.
* **Context management**: Seamless conversations with the GPT model by maintaining message history across CLI calls.
* **Sliding window history**: Automatically trims conversation history while maintaining context to stay within token
  limits.
* **Custom context from local files**: Provide custom context through piping for GPT model reference during
  conversation.
* **Custom chat models**: Use a custom chat model by specifying the model name with the `-m` or `--model` flag.
* **Viper integration**: Robust configuration management.

## Installation

For a quick and easy installation without compiling, you can directly download the pre-built binary for your operating
system and architecture:

### Apple M1 chips

```shell
curl -L -o chatgpt https://github.com/kardolus/chatgpt-cli/releases/download/v1.0.3/chatgpt-darwin-arm64 && chmod +x chatgpt && sudo mv chatgpt /usr/local/bin/
```

### macOS Intel chips

```shell
curl -L -o chatgpt https://github.com/kardolus/chatgpt-cli/releases/download/v1.0.3/chatgpt-darwin-amd64 && chmod +x chatgpt && sudo mv chatgpt /usr/local/bin/
```

### Linux (amd64)

```shell
curl -L -o chatgpt https://github.com/kardolus/chatgpt-cli/releases/download/v1.0.3/chatgpt-linux-amd64 && chmod +x chatgpt && sudo mv chatgpt /usr/local/bin/
```

### Linux (arm64)

```shell
curl -L -o chatgpt https://github.com/kardolus/chatgpt-cli/releases/download/v1.0.3/chatgpt-linux-arm64 && chmod +x chatgpt && sudo mv chatgpt /usr/local/bin/
```

### Windows (amd64)

Download the binary
from [this link](https://github.com/kardolus/chatgpt-cli/releases/download/v1.0.3/chatgpt-windows-amd64.exe) and add it
to your PATH.

Choose the appropriate command for your system, which will download the binary, make it executable, and move it to your
/usr/local/bin directory (or %PATH% on Windows) for easy access.

## Getting Started

1. Set the `OPENAI_API_KEY` environment variable to
   your [ChatGPT secret key](https://platform.openai.com/account/api-keys). To set the environment variable, you can add
   the following line to your shell profile (e.g., ~/.bashrc, ~/.zshrc, or ~/.bash_profile), replacing your_api_key with
   your actual key:

```shell
export OPENAI_API_KEY="your_api_key"
```

2. To enable history tracking across CLI calls, create a ~/.chatgpt-cli directory using the command:

```shell
mkdir -p ~/.chatgpt-cli
```

With this directory in place, the CLI will automatically manage message history for seamless conversations with the GPT
model. The history acts as a sliding window, maintaining a maximum of `4096` tokens to ensure optimal performance and
interaction quality.

3. Try it out:

```shell
chatgpt what is the capital of the Netherlands
```

4. To use the pipe feature, create a text file containing some context. For example, create a file named context.txt
   with the following content:

```shell
Kya is a playful dog who loves swimming and playing fetch.
```

Then, use the pipe feature to provide this context to ChatGPT:

```shell
cat context.txt | chatgpt "What kind of toy would Kya enjoy?"
```

## Development

To start developing, set the `OPENAI_API_KEY` environment variable to
your [ChatGPT secret key](https://platform.openai.com/account/api-keys). Follow these steps for running tests and
building the application:

1. Run the tests using the following scripts:

For unit tests, run:

```shell
./scripts/unit.sh
```

For integration tests, run:

```shell
./scripts/integration.sh
```

To run all tests, use:

```shell
./scripts/all-tests.sh
```

2. Build the app using the installation script:

```shell
./scripts/install.sh
```

3. After a successful build, test the application with the following command:

```shell
./bin/chatgpt what type of dog is a Jack Russel?
```

4. As mentioned before, to enable history tracking across CLI calls, create a ~/.chatgpt-cli directory using the
   command:

```shell
mkdir -p ~/.chatgpt-cli
```

With this directory in place, the CLI will automatically manage message history for seamless conversations with the GPT
model. The history acts as a sliding window, maintaining a maximum of 4096 tokens to ensure optimal performance and
interaction quality.

For more options, see:

```shell
./bin/chatgpt --help
```

## Reporting Issues and Contributing

If you encounter any issues or have suggestions for improvements,
please [submit an issue](https://github.com/kardolus/chatgpt-cli/issues/new) on GitHub. We appreciate your feedback and
contributions to help make this project better.

## Useful Links

* [ChatGPT API Documentation](https://platform.openai.com/docs/introduction/overview)
* [OpenAI API Reference](https://platform.openai.com/docs/api-reference/introduction)
* [Key Usage Dashboard](https://platform.openai.com/account/usage)