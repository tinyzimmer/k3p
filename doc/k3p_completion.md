## k3p completion

Generate completion script

### Synopsis

To load completions:

Bash:

$ source <(k3p completion bash)

# To load completions for each session, execute once:
Linux:
  $ k3p completion bash > /etc/bash_completion.d/k3p
MacOS:
  $ k3p completion bash > /usr/local/etc/bash_completion.d/k3p

Zsh:

# If shell completion is not already enabled in your environment you will need
# to enable it.  You can execute the following once:

$ echo "autoload -U compinit; compinit" >> ~/.zshrc

# To load completions for each session, execute once:
$ k3p completion zsh > "${fpath[1]}/_k3p"

# You will need to start a new shell for this setup to take effect.

Fish:

$ k3p completion fish | source

# To load completions for each session, execute once:
$ k3p completion fish > ~/.config/fish/completions/k3p.fish


```
k3p completion [bash|zsh|fish|powershell]
```

### Options

```
  -h, --help   help for completion
```

### Options inherited from parent commands

```
      --cache-dir string   Override the default location for cached k3s assets (default "/home/<user>/.k3p/cache")
      --tmp-dir string     Override the default tmp directory (default "/tmp")
  -v, --verbose            Enable verbose logging
```

### SEE ALSO

* [k3p](k3p.md)	 - k3p is a k3s packaging and delivery utility

