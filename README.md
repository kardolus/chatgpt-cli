# ChatGPT CLI

This project showcases an implementation of ChatGPT clients with streaming support in a Command-Line Interface (CLI)
environment, demonstrating its practicality and effectiveness.

![a screenshot](resources/recording.gif)

## Features

* Interactive streaming mode for real-time interaction with the GPT model.
* Query mode for single input-output interactions with the GPT model.
* Viper integration for robust configuration management.

## Development

To start developing, set the `OPENAI_API_KEY` environment variable to
your [ChatGPT secret key](https://platform.openai.com/account/api-keys). Follow these steps for running tests and
building the application:

1. Run the unit tests using the following script:

```shell
./scripts/unit.sh
```

2. Build the app using the installation script:

```shell
./scripts/install.sh
```

3. After a successful build, test the application with the following command:

```shell
./bin/chatgpt what type of dog is a Jack Russel?
```

For more options, see:

```shell
./bin/chatgpt --help
```

## Up Next

* Maintain context across multiple calls to ChatGPT.
* Reset the context with a CLI command.

## Useful Links

* [ChatGPT API Documentation](https://platform.openai.com/docs/introduction/overview)
* [Key Usage Dashboard](https://platform.openai.com/account/usage)