# Install Client

- **Install from release for Linux**

1. Download [**df-client-0.0.1.linux-amd64.tar.gz**](https://github.com/alibaba/Dragonfly/raw/master/dist/df-client-0.0.1.linux-amd64.tar.gz) , other versions in 
**[here](https://github.com/alibaba/Dragonfly/blob/master/CHANGELOG.md)**

2. `tar xzvf df-client-0.0.1.linux-amd64.tar.gz -C xxx`, "xxx" is a directory path.

3. Set environment variable named PATH: `PATH=$PATH:xxx/df-client`

- **Install from source code for Linux**

requirements: golang1.7+ and the go cmd must be in environment variable named PATH.

1. `cd source_dir/build/client`,source_dir is the directory where the source code is located.

2. `./configure --prefix=xxx`, --prefix=xxx specifies the installation directory, this cmd param is optional and current dir will be used if --prefix not be specified.

3. `make`

4. `make install`,you can execute `make package` to generate installation package.

5. `make clean`

6. Set environment variable named PATH: `PATH=$PATH:xxx/df-client`
