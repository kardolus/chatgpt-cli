# ChatGPT CLI

![Test Workflow](https://github.com/kardolus/chatgpt-cli/actions/workflows/test.yml/badge.svg?branch=main)

ChatGPT CLI offers a robust interface for interacting with ChatGPT models directly from the command line, equipped with
streaming and advanced configurability.

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
- [Additional Resources](#additional-resources)

## Features

* **Streaming mode**: Real-time interaction with the GPT model.
* **Query mode**: Single input-output interactions with the GPT model.
* **Interactive mode**: The interactive mode allows for a more conversational experience with the model. Exit
  interactive mode by simply typing 'exit'.
* **Thread-based context management**: Enjoy seamless conversations with the GPT model with individualized context for
  each thread, much like your experience on the OpenAI website. Each unique thread has its own history, ensuring
  relevant and coherent responses across different chat instances.
* **Sliding window history**: To stay within token limits, the chat history automatically trims while still preserving
  the necessary context.
* **Custom context from any source**: You can provide the GPT model with a custom context during conversation. This
  context can be piped in from any source, such as local files, standard input, or even another program. This
  flexibility allows the model to adapt to a wide range of conversational scenarios.
* **Model listing**: Access a list of available models using the `-l` or `--list-models` flag.
* **Advanced configuration options**: The CLI supports a layered configuration system where settings can be specified
  through default values, a `config.yaml` file, and environment variables. For quick adjustments, use the `--set-model`
  and `--set-max-tokens` flags. To verify your current settings, use the `--config` or `-c` flag. The newly
  added `omit_history` configuration option adds another layer of customization to your user experience.
* **Availability Note**: This CLI supports both gpt-4 and gpt-3.5-turbo models. However, the specific ChatGPT model used
  on chat.openai.com may not be available via the OpenAI API.

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
curl -L -o chatgpt https://github.com/kardolus/chatgpt-cli/releases/download/v1.3.4/chatgpt-darwin-arm64 && chmod +x chatgpt && sudo mv chatgpt /usr/local/bin/
```

#### macOS Intel chips

```shell
curl -L -o chatgpt https://github.com/kardolus/chatgpt-cli/releases/download/v1.3.4/chatgpt-darwin-amd64 && chmod +x chatgpt && sudo mv chatgpt /usr/local/bin/
```

#### Linux (amd64)

```shell
curl -L -o chatgpt https://github.com/kardolus/chatgpt-cli/releases/download/v1.3.4/chatgpt-linux-amd64 && chmod +x chatgpt && sudo mv chatgpt /usr/local/bin/
```

#### Linux (arm64)

```shell
curl -L -o chatgpt https://github.com/kardolus/chatgpt-cli/releases/download/v1.3.4/chatgpt-linux-arm64 && chmod +x chatgpt && sudo mv chatgpt /usr/local/bin/
```

#### Windows (amd64)

Download the binary
from [this link](https://github.com/kardolus/chatgpt-cli/releases/download/v1.3.4/chatgpt-windows-amd64.exe) and add it
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

   Once this directory is in place, the CLI automatically manages the message history for each "thread" you converse
   with. The history operates like a sliding window, maintaining context up to a configurable token maximum. This
   ensures a balance between maintaining conversation context and achieving optimal performance.

   By default, if a specific thread is not provided by the user, the CLI uses the default thread and stores the history
   at `~/.chatgpt-cli/history/default.json`. You can find more details about how to configure the `thread` parameter in
   the
   [Configuration](#configuration) section of this document.

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

7. For more options, see:

   ```shell
   chatgpt --help
   ```

## Configuration

The ChatGPT CLI adopts a three-tier configuration strategy, with different levels of precedence assigned to default
values, the `config.yaml` file, and environment variables, in that respective order.

Configuration variables:

| Variable            | Description                                                                                                                                            | Default                        |
|---------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------|--------------------------------|
| `name`              | The prefix for environment variable overrides.                                                                                                         | 'openai'                       |
| `api_key`           | Your OpenAI API key.                                                                                                                                   | (none for security)            |
| `model`             | The GPT model used by the application.                                                                                                                 | 'gpt-3.5-turbo'                |
| `max_tokens`        | The maximum number of tokens that can be used in a single API call.                                                                                    | 4096                           |
| `role`              | The system role                                                                                                                                        | 'You are a helpful assistant.' |
| `temperature`       | What sampling temperature to use, between 0 and 2. Higher values make the output more random; lower values make it more focused and deterministic.     | 1.0                            |
| `frequency_penalty` | Number between -2.0 and 2.0. Positive values penalize new tokens based on their existing frequency in the text so far.                                 | 0.0                            |
| `top_p`             | An alternative to sampling with temperature, called nucleus sampling, where the model considers the results of the tokens with top_p probability mass. | 1.0                            |
| `presence_penalty`  | Number between -2.0 and 2.0. Positive values penalize new tokens based on whether they appear in the text so far.                                      | 0.0                            |
| `thread`            | The name of the current chat thread. Each unique thread name has its own context.                                                                      | 'default'                      |
| `omit_history`      | If true, the chat history will not be used to provide context for the GPT model.                                                                       | false                          |
| `url`               | The base URL for the OpenAI API.                                                                                                                       | 'https://api.openai.com'       |
| `completions_path`  | The API endpoint for completions.                                                                                                                      | '/v1/chat/completions'         |
| `models_path`       | The API endpoint for accessing model information.                                                                                                      | '/v1/models'                   |

The defaults can be overridden by providing your own values in the user configuration file,
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
OPENAI_OMIT_HISTORY=true chatgpt what is the capital of Denmark?
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

4. As mentioned previously, the ChatGPT CLI supports tracking conversation history across CLI calls. This feature
   creates a seamless and conversational experience with the GPT model, as the history is utilized as context in
   subsequent interactions.

   To enable this feature, you need to create a `~/.chatgpt-cli` directory using the command:

    ```shell
    mkdir -p ~/.chatgpt-cli
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
* [OpenAI Pricing Page](https://openai.com/pricing)

## Additional Resources

* ["Summarize any text instantly with a single shortcut"](https://medium.com/@kardolus/summarize-any-text-instantly-with-a-single-shortcut-582551bcc6e2)
  on Medium: Dive deep into the capabilities of this CLI tool with this detailed walkthrough.
* [Join the conversation](https://www.reddit.com/r/ChatGPT/comments/14ip6pm/summarize_any_text_instantly_with_a_single/)
  on Reddit: Discuss the tool, ask questions, and share your experiences with our growing community.

Thank you for using ChatGPT CLI!
