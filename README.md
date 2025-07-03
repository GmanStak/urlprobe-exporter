# urlprobe-exporter
参数说明：
“-config”：
     urls：指定url地址信息，包括需要探测的url地址和解析的信息
     settings：指定访问url地址探测周期，返回code值的时间，和url探测超时时间
“-auth”：
     访问/metrics指标需要的认证信息
“-addr”：监听端口，默认：9119
# Linux Service启动文件
# cat urlprobe-exporter.service
[Unit]
Description=Url Probe
After=network.target

[Service]
WorkingDirectory=/usr/local/urlprobe
ExecStart=/usr/local/urlprobe/urlprobe-exporter
Restart=always

[Install]
WantedBy=multi-user.target
