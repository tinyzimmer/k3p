# Example with Variables

This example extends on the simpler [whoami](../whoami) example, except it utilizes
the variable and templating functionality for accepting user input at installation.

To build the example:

```bash
# k3p build will use k3p.yaml automatically if it exists in the current working directory.
# otherwise you can specify the path to one with the --config flag.
$ k3p build
# ...
# ...
```

When the user goes to install the package, they will be prompted for input, or can alternatively use the `--set` flag to `install`.

```bash
$ k3p install package.tar --docker
2020/12/13 20:08:47  [INFO]     Loading the archive
2020/12/13 20:08:47  [INFO]     Creating docker network vibrant_leakey
2020/12/13 20:08:48  [INFO]     Creating docker volume vibrant-leakey-server-0
Please provide a value for dnsName [localhost]: 
Please provide a value for traefikDisabled [false]: 
# ...
# ...
```

The variable functionality is limited to strings and requires default values be set. For complex templating it is better to
embed a helm chart and use the variables for simple substitutions on the values passed to that chart. You can see an example of this
in the [kvdi example](../kvdi).