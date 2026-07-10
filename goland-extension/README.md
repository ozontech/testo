# GoLand Testo

GoLand extension for Go testing framework testo.

It provides gutter actions to run (and debug) either specific suite test or whole suite.

It automatically determines a suite caller (a regular test which calls `testo.RunSuite`)
and runs it using GoLand's native Go Test run configuration.

The extension supports suite callers with default, aliased and dot imports of `github.com/ozontech/testo`.
It also detects direct suite arguments like `Suite{}`, `&Suite{}` and `new(Suite)`.

## Install from source

Ensure you have installed:

- [GoLand itself](https://www.jetbrains.com/go/download/)
- [JDK 21](https://adoptium.net/temurin/releases/)

Then, do the rest:

```bash
# Clone the repository
git clone https://github.com/ozontech/testo.git
cd testo/goland-extension

# Build extension
./gradlew buildPlugin
```

You will get a `testo-goland-extension-0.1.0.zip` file in `build/distributions`.

You can install it using the "Install Plugin from Disk..." action in GoLand:

```text
Settings | Plugins | Gear icon | Install Plugin from Disk...
```

Point GoLand to the `.zip` file and restart the IDE when prompted.
