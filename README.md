# pnr-dev-stack

This repo contains some go code that acts as the Intention Loop controller ,in the PnR model and it demonstrates the PnR model of computing using three seperate nodes or design chunks .


If you are not familiar with go , you can follow the quick installation steps here : 


## Quick Note on Go Installation

Before trying out this example, you'll need Go installed on your system. If you don't have Go yet:

### For macOS:
```bash
# Using Homebrew
brew install go

# Or download from official site
# Visit https://go.dev/dl/ and download the .pkg installer
```

### For Linux:
```bash
# Ubuntu/Debian
sudo apt update
sudo apt install golang-go

# Fedora
sudo dnf install golang
```

### For Windows:
1. Visit https://go.dev/dl/
2. Download the Windows MSI installer
3. Run the installer and follow the prompts

### Verify Installation:
```bash
go version
```

### Setting Up Go Workspace:
Go modules (introduced in Go 1.11) make dependency management much easier. Our example uses modules, so you don't need to worry about GOPATH settings.

If you're new to Go, don't worry - our example uses basic Go features, and the main logic is easy to follow even if you're not a Go expert. The Go code primarily:
- Reads JSON configuration
- Executes shell scripts
- Monitors file changes
- Manages execution states

These are common patterns that would be similar in other languages like Python or Node.js.

---

