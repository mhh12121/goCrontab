# goCrontab
Demo-mainly for practising channel and time package

To run 
```
./goCron.go [port]
```
## Data Structure
```
//this is for json
Task{
    id string
    cmd string
    args []string
    interval int
}
```
then you can simulate sending data to this port, data format is like above
Now only support **POST** and **DELETE** to add or remove cronTab