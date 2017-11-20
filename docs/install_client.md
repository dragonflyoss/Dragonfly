# Install Client

- **Install From Latest Package**

  - Download [df-client.linux-amd64.tar.gz](../package/df-client.linux-amd64.tar.gz) 
  - `tar xzvf df-client.linux-amd64.tar.gz -C xxx`, "xxx" is installation directory.
  - Set environment variable named PATH: `PATH=$PATH:xxx/df-client`

- **Install From Source Code**

  *Requirements: go1.7+ and the go cmd must be in environment variable named PATH.*

  - `cd source_dir/build/client`,source_dir is the directory where the source code is located.

  - `./configure --prefix=xxx`, --prefix=xxx specifies the installation directory, this cmd param is optional and current dir will be used if --prefix not be specified.

  - `make`

  - `make install`,you can execute `make package` to generate installation package.

  - `make clean`

  - Set environment variable named PATH: `PATH=$PATH:xxx/df-client`
