# remote-grep

## introduce
可以同时grep多台机器

## config
```yaml
user: xxxxx
password: xxxx
file: 
  log: xxx.log
  warn: xxx*.warn(可以配置*进行多个文件grep)
test:
  - xx.xx.xx.xx

online:
  - xx.xx.xx

```

mkdir ~/.remote && mv example.yaml ~/.remote/


## usage 
1、remote-grep xxx {project}.{env}.{file} 这个命令可以在任意路径下执行
2、remote-grep xxx xxxx xxxx {project}.{env}.{file} 支持多个条件grep

ps: project 是配置文件的文件名，env是配置文件中节点名可以在里边配不同环境的机器集群，file是file节点中配置的文件路径
