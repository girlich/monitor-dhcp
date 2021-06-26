all:

monitor-dhcp: monitor-dhcp.go
	go build $<

