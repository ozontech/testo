# VSCode Testo

VSCode extension for Go testing framework testo.

![CodeLens provided by this extension](./example.png)

It provides CodeLens to run (and debug) either specific suite test or whole suite.

It automatically determines a suite caller (a regular test which calls `testo.RunSuite`)
and runs it in terminal or debug REPL.

## Install from source

Ensure you have installed:

- [Visual Studio Code itself](https://code.visualstudio.com/download)
- [microsoft/vscode-vsce](https://github.com/microsoft/vscode-vsce)
- [Node.js + NPM](https://nodejs.org/en/download)

Then, do the rest:

```bash
# Clone the repository
git clone https://github.com/ozontech/testo.git
cd testo/vscode-extension

# Install dependencies
npm i

# Build extension
vsce package
```

You will get an `testo-1.0.0.vsix` file in the same directory.

You can install from a command-line with:

```bash
code --install-extension testo-1.0.0.vsix
```

Or using the "Install from VSIX" command in the Extensions view command dropdown,
or the "Extensions: Install from VSIX" command in the Command Palette, point to the `.vsix` file.
