## dfdaemon gen-ca

generate CA files, including ca.key and ca.crt

### Synopsis

generate CA files, including ca.key and ca.crt

```
dfdaemon gen-ca [flags]
```

### Options

```
      --cert-output string         destination path of generated ca.crt file (default "/tmp/ca.crt")
      --common-name string         subject common name of the certificate, if not specified, the hostname will be used
      --expire-duration duration   expire duration of the certificate (default 87600h0m0s)
  -h, --help                       help for gen-ca
      --key-output string          destination path of generated ca.key file (default "/tmp/ca.key")
      --overwrite                  whether to overwrite the existing CA files
```

### SEE ALSO

* [dfdaemon](dfdaemon.md)	 - The dfdaemon is a proxy that intercepts image download requests.

