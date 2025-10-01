# ChatGPT CLI

![Test Workflow](https://github.com/kardolus/chatgpt-cli/actions/workflows/test.yml/badge.svg?branch=main) [![Public Backlog](https://img.shields.io/badge/public%20backlog-808080)](https://github.com/users/kardolus/projects/2)

**Tested and Compatible with OpenAI ChatGPT, Azure OpenAI Service, Perplexity AI, Llama and 302.AI!**

ChatGPT CLI provides a powerful command-line interface for seamless interaction with ChatGPT models via OpenAI and
Azure, featuring streaming capabilities and extensive configuration options.

![a screenshot](cmd/chatgpt/resources/vhs.gif)

## Table of Contents

- [Features](#features)
    - [Prompt Support](#prompt-support)
        - [Using the prompt flag](#using-the---prompt-flag)
        - [Example](#example)
        - [Explore More Prompts](#explore-more-prompts)
    - [MCP Support](#mcp-support)
        - [Overview](#overview)
        - [Examples](#examples)
        - [Default Version Behavior](#default-version-behavior)
        - [Handling MCP Replies](#handling-mcp-replies)
        - [Config](#config)
- [Installation](#installation)
    - [Using Homebrew (macOS)](#using-homebrew-macos)
    - [Direct Download](#direct-download)
        - [Apple Silicon](#apple-silicon)
        - [macOS Intel chips](#macos-intel-chips)
        - [Linux (amd64)](#linux-amd64)
        - [Linux (arm64)](#linux-arm64)
        - [Linux (386)](#linux-386)
        - [FreeBSD (amd64)](#freebsd-amd64)
        - [FreeBSD (arm64)](#freebsd-arm64)
        - [Windows (amd64)](#windows-amd64)
- [Getting Started](#getting-started)
- [Configuration](#configuration)
    - [General Configuration](#general-configuration)
    - [LLM Specific Configuration](#llm-specific-configuration)
    - [Custom Config and Data Directory](#custom-config-and-data-directory)
        - [Example for Custom Directories](#example-for-custom-directories)
        - [Variables for interactive mode](#variables-for-interactive-mode)
    - [Switching Between Configurations with --target](#switching-between-configurations-with---target)
    - [Azure Configuration](#azure-configuration)
    - [Perplexity Configuration](#perplexity-configuration)
    - [302 AI Configuration](#302ai-configuration)
    - [Command-Line Autocompletion](#command-line-autocompletion)
        - [Enabling Autocompletion](#enabling-autocompletion)
        - [Persistent Autocompletion](#persistent-autocompletion)
- [Markdown Rendering](#markdown-rendering)
- [Development](#development)
    - [Using the Makefile](#using-the-makefile)
    - [Testing the CLI](#testing-the-cli)
- [Reporting Issues and Contributing](#reporting-issues-and-contributing)
- [Uninstallation](#uninstallation)
- [Useful Links](#useful-links)
- [Additional Resources](#additional-resources)

## Features

* **Streaming mode**: Real-time interaction with the GPT model.
* **Query mode**: Single input-output interactions with the GPT model.
* **Interactive mode**: The interactive mode allows for a more conversational experience with the model. Prints the
  token usage when combined with query mode.
* **Thread-based context management**: Enjoy seamless conversations with the GPT model with individualized context for
  each thread, much like your experience on the OpenAI website. Each unique thread has its own history, ensuring
  relevant and coherent responses across different chat instances.
* **Sliding window history**: To stay within token limits, the chat history automatically trims while still preserving
  the necessary context. The size of this window can be adjusted through the `context-window` setting.
* **Custom context from any source**: You can provide the GPT model with a custom context during conversation. This
  context can be piped in from any source, such as local files, standard input, or even another program. This
  flexibility allows the model to adapt to a wide range of conversational scenarios.
* **Support for images**: Upload an image or provide an image URL using the `--image` flag. Note that image support may
  not be available for all models. You can also pipe an image directly: `pngpaste - | chatgpt "What is this photo?"`
* **Generate images**: Use the `--draw` and `--output` flags to generate an image from a prompt (requires image-capable
  models like `gpt-image-1`).
* **Edit images**: Use the `--draw` flag with `--image` and `--output` to modify an existing image using a prompt (
  e.g., "add sunglasses to the cat"). Supported formats: PNG, JPEG, and WebP.
* **Audio support**: You can upload audio files using the `--audio` flag to ask questions about spoken content.
  This feature is compatible only with audio-capable models like gpt-4o-audio-preview. Currently, only `.mp3` and `.wav`
  formats are supported.
* **Transcription support**: You can also use the `--transcribe` flag to generate a transcript of the uploaded audio.
  This uses OpenAI’s transcription endpoint (compatible with models like gpt-4o-transcribe) and supports a wider range
  of formats, including `.mp3`, `.mp4`, `.mpeg`, `.mpga`, `.m4a`, `.wav`, and `.webm`.
* **Text-to-speech support**: Use the `--speak` and `--output` flags to convert text to speech (works with models like
  `gpt-4o-mini-tts`).
  If you have `afplay` installed (macOS), you can even chain playback like this:
    ```shell
    chatgpt --speak "convert this to audio" --output test.mp3 && afplay test.mp3
    ```
* **Model listing**: Access a list of available models using the `-l` or `--list-models` flag.
* **Advanced configuration options**: The CLI supports a layered configuration system where settings can be specified
  through default values, a `config.yaml` file, and environment variables. For quick adjustments,
  various `--set-<value>` flags are provided. To verify your current settings, use the `--config` or `-c` flag.

### Prompt Support

We’re excited to introduce support for prompt files with the `--prompt` flag in **version 1.7.1**! This feature
allows you to provide a rich and detailed context for your conversations directly from a file.

#### Using the `--prompt` Flag

The `--prompt` flag lets you specify a file containing the initial context or instructions for your ChatGPT
conversation. This is especially useful when you have detailed instructions or context that you want to reuse across
different conversations.

To use the `--prompt` flag, pass the path of your prompt file like this:

```shell
chatgpt --prompt path/to/your/prompt.md "Use a pipe or provide a query here"
```

The contents of `prompt.md` will be read and used as the initial context for the conversation, while the query you
provide directly will serve as the specific question or task you want to address.

#### Example

Here’s a fun example where you can use the output of a `git diff` command as a prompt:

```shell
git diff | chatgpt --prompt ../prompts/write_pull-request.md
```

In this example, the content from the `write_pull-request.md` prompt file is used to guide the model's response based on
the diff data from `git diff`.

#### Explore More Prompts

For a variety of ready-to-use prompts, check out this [awesome prompts repository](https://github.com/kardolus/prompts).
These can serve as great starting points or inspiration for your own custom prompts!

Here’s the updated README section for MCP Support, placed after the ### Prompt Support section you shared:

### MCP Support

We’re excited to introduce Model Context Protocol (MCP) support in version 1.8.3+, allowing you to enrich your chat
sessions with structured, live data. For now, this feature is limited to Apify integrations.

#### Overview

MCP enables the CLI to call external plugins — like Apify actors — and inject their responses into the chat context
before your actual query is sent. This is useful for fetching weather, scraping Google Maps, or summarizing PDFs.

You can use either `--param` (for individual key=value pairs) or `--params` (for raw JSON).

#### Examples

Using `--param` flags:

```shell
chatgpt --mcp apify/epctex~weather-scraper \
    --param locations='["Brooklyn"]' \
    --param language=en \
    --param forecasts=true \
    "what should I wear today"
```

Using a single `--params` flag:

```shell
chatgpt --mcp apify/epctex~weather-scraper \
    --params '{"locations": ["Brooklyn"], "language": "en", "forecasts": true}' \
    "what should I wear today"
```

#### Default Version Behavior

If no version is specified, `@latest` is assumed:

```shell
chatgpt --mcp apify/user~weather
```

is equivalent to:

```shell
chatgpt --mcp apify/user~weather@latest
```

#### Handling MCP Replies

Responses from MCP plugins are automatically injected into the conversation thread as context. You can use MCP in two
different modes:

1. MCP-only mode (Context Injection Only)

    ```shell
    chatgpt --mcp apify/epctex~weather-scraper --param location=Brooklyn
    ```

    * Fetches live data
    * Injects it into the current thread
    * Does not trigger a GPT completion
    * CLI prints a confirmation

2. MCP + Query mode (Context + Completion)

    ```shell
    chatgpt --mcp apify/epctex~weather-scraper --param location=Brooklyn "What should I wear today?"
    ```

    * Fetches and injects MCP data
    * Immediately sends your query to GPT
    * Returns the assistant’s response

#### Config

You’ll need to set the `APIFY_API_KEY` as an environment variable or config value

Example:

```shell
export APIFY_API_KEY=your-api-key
```

## Installation

### Using Homebrew (macOS)

You can install chatgpt-cli using Homebrew:

```shell
brew tap kardolus/chatgpt-cli && brew install chatgpt-cli
```

### Direct Download

For a quick and easy installation without compiling, you can directly download the pre-built binary for your operating
system and architecture:

#### Apple Silicon

```shell
curl -L -o chatgpt https://github.com/kardolus/chatgpt-cli/releases/latest/download/chatgpt-darwin-arm64 && chmod +x chatgpt && sudo mv chatgpt /usr/local/bin/
```

#### macOS Intel chips

```shell
curl -L -o chatgpt https://github.com/kardolus/chatgpt-cli/releases/latest/download/chatgpt-darwin-amd64 && chmod +x chatgpt && sudo mv chatgpt /usr/local/bin/
```

#### Linux (amd64)

```shell
curl -L -o chatgpt https://github.com/kardolus/chatgpt-cli/releases/latest/download/chatgpt-linux-amd64 && chmod +x chatgpt && sudo mv chatgpt /usr/local/bin/
```

#### Linux (arm64)

```shell
curl -L -o chatgpt https://github.com/kardolus/chatgpt-cli/releases/latest/download/chatgpt-linux-arm64 && chmod +x chatgpt && sudo mv chatgpt /usr/local/bin/
```

#### Linux (386)

```shell
curl -L -o chatgpt https://github.com/kardolus/chatgpt-cli/releases/latest/download/chatgpt-linux-386 && chmod +x chatgpt && sudo mv chatgpt /usr/local/bin/
```

#### FreeBSD (amd64)

```shell
curl -L -o chatgpt https://github.com/kardolus/chatgpt-cli/releases/latest/download/chatgpt-freebsd-amd64 && chmod +x chatgpt && sudo mv chatgpt /usr/local/bin/
```

#### FreeBSD (arm64)

```shell
curl -L -o chatgpt https://github.com/kardolus/chatgpt-cli/releases/latest/download/chatgpt-freebsd-arm64 && chmod +x chatgpt && sudo mv chatgpt /usr/local/bin/
```

#### Windows (amd64)

Download the binary
from [this link](https://github.com/kardolus/chatgpt-cli/releases/latest/download/chatgpt-windows-amd64.exe) and add it
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

   If you want the CLI to automatically create a new thread for each session, ensure that the `auto_create_new_thread`
   configuration variable is set to `true`. This will create a unique thread identifier for each interactive session.

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

The ChatGPT CLI adopts a four-tier configuration strategy, with different levels of precedence assigned to flags,
environment variables, a config.yaml file, and default values, in that respective order:

1. Flags: Command-line flags have the highest precedence. Any value provided through a flag will override other
   configurations.
2. Environment Variables: If a setting is not specified by a flag, the corresponding environment variable (prefixed with
   the name field from the config) will be checked.
3. Config file (config.yaml): If neither a flag nor an environment variable is set, the value from the config.yaml file
   will be used.
4. Default Values: If no value is specified through flags, config.yaml, or environment variables, the CLI will fall back
   to its built-in default values.

### General Configuration

| Variable                 | Description                                                                                                                                                                                           | Default                   |
|--------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------------------------|
| `name`                   | The prefix for environment variable overrides.                                                                                                                                                        | 'openai'                  |
| `thread`                 | The name of the current chat thread. Each unique thread name has its own context.                                                                                                                     | 'default'                 |
| `target`                 | Load configuration from config._target_.yaml                                                                                                                                                          | ''                        |
| `omit_history`           | If true, the chat history will not be used to provide context for the GPT model.                                                                                                                      | false                     |
| `command_prompt`         | The command prompt in interactive mode. Should be single-quoted.                                                                                                                                      | '[%datetime] [Q%counter]' |
| `output_prompt`          | The output prompt in interactive mode. Should be single-quoted.                                                                                                                                       | ''                        |
| `command_prompt_color`   | The color of the command_prompt in interactive mode. Supported colors: "red", "green", "blue", "yellow", "magenta".                                                                                   | ''                        |
| `output_prompt_color`    | The color of the output_prompt in interactive mode. Supported colors: "red", "green", "blue", "yellow", "magenta".                                                                                    | ''                        |
| `auto_create_new_thread` | If set to `true`, a new thread with a unique identifier (e.g., `int_a1b2`) will be created for each interactive session. If `false`, the CLI will use the thread specified by the `thread` parameter. | `false`                   |
| `track_token_usage`      | If set to true, displays the total token usage after each query in --query mode, helping you monitor API usage.                                                                                       | `false`                   |
| `debug`                  | If set to true, prints the raw request and response data during API calls, useful for debugging.                                                                                                      | `false`                   |
| `skip_tls_verify`        | If set to true, skips TLS certificate verification, allowing insecure HTTPS requests.                                                                                                                 | `false`                   |
| `multiline`              | If set to true, enables multiline input mode in interactive sessions.                                                                                                                                 | `false`                   |
| `role_file`              | Path to a file that overrides the system role (role).                                                                                                                                                 | ''                        |
| `prompt`                 | Path to a file that provides additional context before the query.                                                                                                                                     | ''                        |
| `image`                  | Local path or URL to an image used in the query.                                                                                                                                                      | ''                        |
| `audio`                  | Path to an audio file (MP3/WAV) used as part of the query.                                                                                                                                            | ''                        |
| `output`                 | Path where synthesized audio is saved when using --speak.                                                                                                                                             | ''                        |
| `transcribe`             | Enables transcription mode. This flags takes the path of an audio file.                                                                                                                               | `false`                   |
| `speak`                  | If true, enables text-to-speech synthesis for the input query.                                                                                                                                        | `false`                   |
| `draw`                   | If true, generates an image from a prompt and saves it to the path specified by `output`. Requires image-capable models.                                                                              | `false`                   |

### LLM-Specific Configuration

| Variable                 | Description                                                                                                                                            | Default                        |
|--------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------|--------------------------------|
| `api_key`                | Your API key.                                                                                                                                          | (none for security)            |
| `auth_header`            | The header used for authorization in API requests.                                                                                                     | 'Authorization'                |
| `auth_token_prefix`      | The prefix to be added before the token in the `auth_header`.                                                                                          | 'Bearer '                      |
| `completions_path`       | The API endpoint for completions.                                                                                                                      | '/v1/chat/completions'         |
| `context_window`         | The memory limit for how much of the conversation can be remembered at one time.                                                                       | 8192                           |
| `effort`                 | Sets the reasoning effort. Used by o1-pro models.                                                                                                      | 'low'                          |
| `frequency_penalty`      | Number between -2.0 and 2.0. Positive values penalize new tokens based on their existing frequency in the text so far.                                 | 0.0                            |
| `image_edits_path`       | The API endpoint for image editing.                                                                                                                    | '/v1/images/edits'             |
| `image_generations_path` | The API endpoint for image generation.                                                                                                                 | '/v1/images/generations'       |
| `max_tokens`             | The maximum number of tokens that can be used in a single API call.                                                                                    | 4096                           |
| `model`                  | The GPT model used by the application.                                                                                                                 | 'gpt-4o'                       |
| `models_path`            | The API endpoint for accessing model information.                                                                                                      | '/v1/models'                   |
| `presence_penalty`       | Number between -2.0 and 2.0. Positive values penalize new tokens based on whether they appear in the text so far.                                      | 0.0                            |
| `responses_path`         | The API endpoint for responses. Used by o1-pro models.                                                                                                 | '/v1/responses'                |
| `role`                   | The system role                                                                                                                                        | 'You are a helpful assistant.' |
| `seed`                   | Sets the seed for deterministic sampling (Beta). Repeated requests with the same seed and parameters aim to return the same result.                    | 0                              |
| `speech_path`            | The API endpoint for text-to-speech synthesis.                                                                                                         | '/v1/audio/transcriptions'     |
| `temperature`            | What sampling temperature to use, between 0 and 2. Higher values make the output more random; lower values make it more focused and deterministic.     | 1.0                            |
| `top_p`                  | An alternative to sampling with temperature, called nucleus sampling, where the model considers the results of the tokens with top_p probability mass. | 1.0                            |
| `transcriptions_path`    | The API endpoint for audio transcription requests.                                                                                                     | '/v1/audio/speech'             |
| `url`                    | The base URL for the OpenAI API.                                                                                                                       | 'https://api.openai.com'       |
| `user_agent`             | The header used for the user agent in API requests.                                                                                                    | 'chatgpt-cli'                  |
| `voice`                  | The voice to use when generating audio with TTS models like gpt-4o-mini-tts.                                                                           | 'nova'                         |

### Custom Config and Data Directory

By default, ChatGPT CLI stores configuration and history files in the `~/.chatgpt-cli` directory. However, you can
easily
override these locations by setting environment variables, allowing you to store configuration and history in custom
directories.

| Environment Variable | Description                                  | Default Location         |
|----------------------|----------------------------------------------|--------------------------|
| `OPENAI_CONFIG_HOME` | Overrides the default config directory path. | `~/.chatgpt-cli`         |
| `OPENAI_DATA_HOME`   | Overrides the default data directory path.   | `~/.chatgpt-cli/history` |

#### Example for Custom Directories

To change the default configuration or data directories, set the appropriate environment variables:

```
export OPENAI_CONFIG_HOME="/custom/config/path"
export OPENAI_DATA_HOME="/custom/data/path"
```

If these environment variables are not set, the application defaults to ~/.chatgpt-cli for configuration files and ~
/.chatgpt-cli/history for history.

### Switching Between Configurations with --target

You can maintain multiple configuration files side by side and switch between them using the `--target` flag. This is
especially useful if you use multiple LLM providers (like OpenAI, Perplexity, Azure, etc.) or have different contexts or
workflows that require distinct settings.

How it Works

When you use the `--target` flag, the CLI loads a config file named:

```shell
config.<target>.yaml
```

For example:

```shell
chatgpt --target perplexity --config
```

This will load:

```shell
~/.chatgpt-cli/config.perplexity.yaml
```

If the --target flag is not provided, the CLI falls back to:

```shell
~/.chatgpt-cli/config.yaml
```

Example Setup

You can maintain the following structure:

```shell
~/.chatgpt-cli/
├── config.yaml # Default (e.g., OpenAI)
├── config.perplexity.yaml # Perplexity setup
├── config.azure.yaml # Azure-specific config
└── config.llama.yaml # LLaMA setup
```

Then switch between them like so:

```shell
chatgpt --target azure "Explain Azure's GPT model differences"
chatgpt --target perplexity "What are some good restaurants in the Red Hook area"
```

Or just use the default:

```shell
chatgpt "What's the capital of Sweden?"
```

CLI and Environment Interaction

* The value of `--target` is never persisted — it must be explicitly passed for each run.
* The config file corresponding to the target is loaded before any environment variable overrides are applied.
* Environment variables still follow the name: field inside the loaded config, so name: perplexity enables
  `PERPLEXITY_API_KEY`.

#### Variables for interactive mode:

- `%date`: The current date in the format `YYYY-MM-DD`.
- `%time`: The current time in the format `HH:MM:SS`.
- `%datetime`: The current date and time in the format `YYYY-MM-DD HH:MM:SS`.
- `%counter`: The total number of queries in the current session.
- `%usage`: The usage in total tokens used (only works in query mode).

The defaults can be overridden by providing your own values in the user configuration file. The structure of this file
mirrors that of the default configuration. For instance, to override
the `model` and `max_tokens` parameters, your file might look like this:

```yaml
model: gpt-3.5-turbo-16k
max_tokens: 4096
```

This alters the `model` to `gpt-3.5-turbo-16k` and adjusts `max_tokens` to `4096`. All other options, such as `url`
, `completions_path`, and `models_path`, can similarly be modified. 

You can also add custom HTTP headers to all API requests. This is useful when working with proxies, API gateways, or services that require additional headers:

```yaml
custom_headers:
  X-Custom-Header: "custom-value"
  X-API-Version: "v2"
  X-Client-ID: "my-client-id"
```

If the user configuration file cannot be accessed or
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

To facilitate convenient adjustments, the ChatGPT CLI provides flags for swiftly modifying the `model`, `thread`
, `context-window` and `max_tokens` parameters in your user configured `config.yaml`. These flags are `--set-model`
, `--set-thread`, `--set-context-window` and `--set-max-tokens`.

For instance, to update the model, use the following command:

```shell
chatgpt --set-model gpt-3.5-turbo-16k
```

This feature allows for rapid changes to key configuration parameters, optimizing your experience with the ChatGPT CLI.

### Azure Configuration

For Azure, you need to configure these, or similar, value

```yaml
name: azure
api_key: <your azure api key>
url: https://<your_resource>.openai.azure.com
completions_path: /openai/deployments/<your_deployment>/chat/completions?api-version=<your_api>
auth_header: api-key
auth_token_prefix: " "
```

You can set the API key either in the config.yaml file as shown above or export it as an environment variable:

```shell
export AZURE_API_KEY=<your_key>
```

### Perplexity Configuration

For Perplexity, you will need something equivelent to the following values:

```yaml
name: perplexity
api_key: <your perplexity api key>
model: sonar
url: https://api.perplexity.ai
```

You can set the API key either in the config.yaml file as shown above or export it as an environment variable:

```shell
export PERPLEXITY_API_KEY=<your_key>
```

You can set the API key either in the `config.yaml` file as shown above or export it as an environment variable:

```shell
export AZURE_API_KEY=<your_key>
```

### 302.AI Configuration

I successfully tested 302.AI with the following values

```yaml
name: ai302 # environment variables cannot start with numbers
api_key: <your 302.AI api key>
url: https://api.302.ai
```

You can set the API key either in the config.yaml file as shown above or export it as an environment variable:

```shell
export AI302_API_KEY=<your_key>
```

### Command-Line Autocompletion

Enhance your CLI experience with our new autocompletion feature for command flags!

#### Enabling Autocompletion

Autocompletion is currently supported for the following shells: Bash, Zsh, Fish, and PowerShell. To activate flag
completion in your current shell session, execute the appropriate command based on your shell:

- **Bash**
    ```bash
    . <(chatgpt --set-completions bash)
    ```
- **Zsh**
    ```zsh
    . <(chatgpt --set-completions zsh)
    ```
- **Fish**
    ```fish
    chatgpt --set-completions fish | source
    ```
- **PowerShell**
    ```powershell
    chatgpt --set-completions powershell | Out-String | Invoke-Expression
    ```

#### Persistent Autocompletion

For added convenience, you can make autocompletion persist across all new shell sessions by adding the appropriate
sourcing command to your shell's startup file. Here are the files typically used for each shell:

- **Bash**: Add to `.bashrc` or `.bash_profile`
- **Zsh**: Add to `.zshrc`
- **Fish**: Add to `config.fish`
- **PowerShell**: Add to your PowerShell profile script

For example, for Bash, you would add the following line to your `.bashrc` file:

```bash
. <(chatgpt --set-completions bash)
```

This ensures that command flag autocompletion is enabled automatically every time you open a new terminal window.

## Markdown Rendering

You can render markdown in real-time using the `mdrender.sh` script, located [here](scripts/mdrender.sh). You'll first
need to
install [glow](https://github.com/charmbracelet/glow).

Example:

```shell
chatgpt write a hello world program in Java | ./scripts/mdrender.sh
```

## Development

To start developing, set the `OPENAI_API_KEY` environment variable to
your [ChatGPT secret key](https://platform.openai.com/account/api-keys).

### Using the Makefile

The Makefile simplifies development tasks by providing several targets for testing, building, and deployment.

* **all-tests**: Run all tests, including linting, formatting, and go mod tidy.
  ```shell 
  make all-tests
  ```
* **binaries**: Build binaries for multiple platforms.
  ```shell 
  make binaries
  ```
* **shipit**: Run the release process, create binaries, and generate release notes.
  ```shell 
  make shipit
  ```
* **updatedeps**: Update dependencies and commit any changes.
  ```shell 
  make updatedeps
  ```

For more available commands, use:

```shell
make help
```

#### Windows build script

```ps1
.\scripts\install.ps1
```

### Testing the CLI

1. After a successful build, test the application with the following command:

    ```shell
    ./bin/chatgpt what type of dog is a Jack Russel?
    ```

2. As mentioned previously, the ChatGPT CLI supports tracking conversation history across CLI calls. This feature
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

* [Amazing Prompts](https://github.com/kardolus/prompts)
* [OpenAI API Reference](https://platform.openai.com/docs/api-reference/chat/create)
* [OpenAI Key Usage Dashboard](https://platform.openai.com/account/usage)
* [OpenAI Pricing Page](https://openai.com/pricing)
* [Perplexity API Reference](https://docs.perplexity.ai/reference/post_chat_completions)
* [Perplexity Key Usage Dashboard](https://www.perplexity.ai/settings/api)
* [Perplexity Models](https://docs.perplexity.ai/docs/model-cards)
* [302.AI API Reference](https://302ai-en.apifox.cn/api-207705102)

## Additional Resources

* ["Summarize any text instantly with a single shortcut"](https://medium.com/@kardolus/summarize-any-text-instantly-with-a-single-shortcut-582551bcc6e2)
  on Medium: Dive deep into the capabilities of this CLI tool with this detailed walkthrough.
* [Join the conversation](https://www.reddit.com/r/ChatGPT/comments/14ip6pm/summarize_any_text_instantly_with_a_single/)
  on Reddit: Discuss the tool, ask questions, and share your experiences with our growing community.

Thank you for using ChatGPT CLI!

<div align="center" style="text-align: center; display: flex; justify-content: center; align-items: center;">
    <a href="#top">
        <img src="https://img.shields.io/badge/Back%20to%20Top-000000?style=for-the-badge&logo=github&logoColor=white" alt="Back to Top">
    </a>
</div>

