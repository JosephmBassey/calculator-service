# calculator-service
# calculator-service

 A simple calculator program that calculate simple math tasks, input is sent through the [Client](https://github.com/JosephmBassey/calculator-client). 
## Clone  the project

```
$ git clone https://github.com/JosephmBassey/calculator-service
$ cd calculator-service
```
## Set the calculator server environment variables.
 Assuming you want the server to run on `0.0.0.0:9081`
```
$ export LISTEN_GRPC=0.0.0.0:9081
```

## build and run the project.
- still in the calculator-service dir.
```
$ make build
$ ./server 
```