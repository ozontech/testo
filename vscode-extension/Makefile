MAKEFLAGS += --always-make

# build extension and install to visual studio code
install: package
	code --install-extension testo.vsix

# build using vsce
package: deps
	vsce package --out testo.vsix

# install npm deps
deps:
	npm i
