# ChatGPT CLI

This project showcases an implementation of a ChatGPT client with streaming support in a Command-Line Interface (CLI)
environment, demonstrating its practicality and effectiveness.

![a screenshot](resources/vhs.gif)

## Table of Contents

- [Features](#features)
- [Installation](#installation)
    - [Using Homebrew (macOS)](#using-homebrew-macos)
    - [Direct Download](#direct-download)
        - [Apple M1 chips](#apple-m1-chips)
        - [macOS Intel chips](#macos-intel-chips)
        - [Linux (amd64)](#linux-amd64)
        - [Linux (arm64)](#linux-arm64)
        - [Windows (amd64)](#windows-amd64)
- [Getting Started](#getting-started)
- [Configuration](#configuration)
- [Development](#development)
- [Reporting Issues and Contributing](#reporting-issues-and-contributing)
- [Uninstallation](#uninstallation)
- [Useful Links](#useful-links)

## Features

* **Streaming mode**: Real-time interaction with the GPT model.
* **Query mode**: Single input-output interactions with the GPT model.
* **Interactive mode**: The interactive mode allows for a more conversational experience with the model. Exit
  interactive mode by simply typing 'exit'.
* **Context management**: Seamless conversations with the GPT model by maintaining message history across CLI calls.
* **Sliding window history**: Automatically trims conversation history while maintaining context to stay within token
  limits.
* **Custom context from local files**: Provide a custom context for the GPT model to reference during the conversation
  by piping it in.
* **Custom chat models**: Use a custom chat model by specifying the model name with the `--set-model` flag. Ensure that
  the model exists in the OpenAI model list.
* **Model listing**: Get a list of available models by using the `-l` or `--list-models` flag.
* **Viper integration**: Robust configuration management.

## Installation

### Using Homebrew (macOS)

You can install chatgpt-cli using Homebrew:

```shell
brew tap kardolus/chatgpt-cli && brew install chatgpt-cli
```

### Direct Download

For a quick and easy installation without compiling, you can directly download the pre-built binary for your operating
system and architecture:

#### Apple M1 chips

```shell
curl -L -o chatgpt https://github.com/kardolus/chatgpt-cli/releases/download/v1.2.0/chatgpt-darwin-arm64 && chmod +x chatgpt && sudo mv chatgpt /usr/local/bin/
```

#### macOS Intel chips

```shell
curl -L -o chatgpt https://github.com/kardolus/chatgpt-cli/releases/download/v1.2.0/chatgpt-darwin-amd64 && chmod +x chatgpt && sudo mv chatgpt /usr/local/bin/
```

#### Linux (amd64)

```shell
curl -L -o chatgpt https://github.com/kardolus/chatgpt-cli/releases/download/v1.2.0/chatgpt-linux-amd64 && chmod +x chatgpt && sudo mv chatgpt /usr/local/bin/
```

#### Linux (arm64)

```shell
curl -L -o chatgpt https://github.com/kardolus/chatgpt-cli/releases/download/v1.2.0/chatgpt-linux-arm64 && chmod +x chatgpt && sudo mv chatgpt /usr/local/bin/
```

#### Windows (amd64)

Download the binary
from [this link](https://github.com/kardolus/chatgpt-cli/releases/download/v1.2.0/chatgpt-windows-amd64.exe) and add it
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

4. To start interactive mode, use the `-i` or `--interactive` flag:

```shell
chatgpt --interactive
```

5. To use the pipe feature, create a text file containing some context. For example, create a file named context.txt
   with the following content:

```shell
Kya is a playful dog who loves swimming and playing fetch.
```

Then, use the pipe feature to provide this context to ChatGPT:

```shell
cat context.txt | chatgpt "What kind of toy would Kya enjoy?"
```

6. To list all available models, use the -l or --list-models flag:

```shell
chatgpt --list-models
```

## Configuration

The ChatGPT CLI adopts a three-tier configuration strategy, with different levels of precedence assigned to default
values, the `config.yaml` file, and environment variables, in that respective order.

The default configuration:

```yaml
name: openai
api_key:
model: gpt-3.5-turbo
max_tokens: 4096
url: https://api.openai.com
completions_path: /v1/chat/completions
models_path: /v1/models
omit_history: false
```

These defaults can be overridden by providing your own values in the user configuration file,
named `.chatgpt-cli/config.yaml`, located in your home directory.

The structure of the user configuration file mirrors that of the default configuration. For instance, to override
the `model` and `max_tokens` parameters, your file might look like this:

```yaml
model: gpt-3.5-turbo-16k
max_tokens: 8192
```

This alters the `model` to `gpt-3.5-turbo-16k` and adjusts `max_tokens` to `8192`. All other options, such as `url`
, `completions_path`, and `models_path`, can similarly be modified. If the user configuration file cannot be accessed or
is missing, the application will resort to the default configuration.

Another way to adjust values without manually editing the configuration file is by using environment variables.
The `name` attribute forms the prefix for these variables. As an example, the `model` can be modified using
the `OPENAI_MODEL` environment variable. Similarly, to disable history during the execution of a command, use:

```shell
OPENAI_OMIT_HISTORY=true chatgpt tell me a joke
```

This approach is especially beneficial for temporary changes or for testing varying configurations.

Moreover, you can use the `--config` or `-c` flag to view the present configuration. This handy feature allows users to
swiftly verify their current settings without the need to manually inspect the configuration files.

```shell
chatgpt --config
```

Executing this command will display the active configuration, including any overrides instituted by environment
variables or the user configuration file.

To facilitate convenient adjustments, the ChatGPT CLI provides two flags for swiftly modifying the `model`
and `max_tokens` parameters in your user configured `config.yaml`. These flags are `--set-model` and `--set-max-tokens`.

For instance, to update the model, use the following command:

```shell
chatgpt --set-model gpt-3.5-turbo-16k
```

This feature allows for rapid changes to key configuration parameters, optimizing your experience with the ChatGPT CLI.

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

For contract tests, run:

```shell
./scripts/contract.sh
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

## Uninstallation

If for any reason you wish to uninstall the ChatGPT CLI application from your system, you can do so by following these
steps:

### Using Homebrew (macOS)

If you installed the CLI using Homebrew you can do:

```shell
brew uninstall chatgpt-cli
```

And to remove the tap:

```shell
brew untap kardolus/chatgpt-cli
```

### MacOS / Linux

If you installed the binary directly, follow these steps:

1. Remove the binary:

    ```shell
    sudo rm /usr/local/bin/chatgpt
    ```

2. Optionally, if you wish to remove the history tracking directory, you can also delete the `~/.chatgpt-cli` directory:

    ```shell
    rm -rf ~/.chatgpt-cli
    ```

### Windows

1. Navigate to the location of the `chatgpt` binary in your system, which should be in your PATH.

2. Delete the `chatgpt` binary.

3. Optionally, if you wish to remove the history tracking, navigate to the `~/.chatgpt-cli` directory (where `~` refers
   to your user's home directory) and delete it.

Please note that the history tracking directory `~/.chatgpt-cli` only contains conversation history and no personal
data. If you have any concerns about this, please feel free to delete this directory during uninstallation.

## Useful Links

* [ChatGPT API Documentation](https://platform.openai.com/docs/introduction/overview)
* [OpenAI API Reference](https://platform.openai.com/docs/api-reference/introduction)
* [Key Usage Dashboard](https://platform.openai.com/account/usage)

Thank you for using ChatGPT CLI!
