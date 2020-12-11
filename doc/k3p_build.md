## k3p build

Build a k3s distribution package

```
k3p build [flags]
```

### Options

```
  -a, --arch string             The architecture to package the distribution for. Only (amd64, arm, and arm64 are supported) (default "amd64")
  -C, --channel string          The release channel to retrieve the version of k3s from (default "stable")
  -c, --config string           An optional config file providing variables to be used at installation
  -E, --eula string             A file containing an End User License Agreement to display to the user upon installing the package
  -e, --exclude strings         Directories to exclude when reading the manifest directory
      --exclude-images          Don't include container images with the final archive
  -h, --help                    help for build
  -I, --image-file string       A file containing a list of extra images to bundle with the archive
  -i, --images strings          A comma separated list of images to include with the archive
      --k3s-version string      A specific k3s version to bundle with the package, overrides --channel (default "latest")
  -m, --manifests stringArray   Directories to scan for kubernetes manifests and charts, defaults to the current directory, can be specified multiple times (default [/home/<user>/devel/k3p])
  -n, --name string             The name to give the package, if not provided one will be generated
  -N, --no-cache                Disable the use of the local cache when downloading assets
  -o, --output string           The file to save the distribution package to (default "/home/<user>/devel/k3p/package.tar")
      --pull-policy string      The pull policy to use when bundling container images (valid options always,never,ifnotpresent [case-insensitive]) (default "always")
  -V, --version string          The version to tag the package (default "latest")
```

### Options inherited from parent commands

```
      --cache-dir string   Override the default location for cached k3s assets (default "/home/<user>/.k3p/cache")
      --tmp-dir string     Override the default tmp directory (default "/tmp")
  -v, --verbose            Enable verbose logging
```

### SEE ALSO

* [k3p](k3p.md)	 - k3p is a k3s packaging and delivery utility

