module github.com/romikb/amneziawg-client-windows

go 1.20

require (
	github.com/lxn/walk v0.0.0-20210112085537-c389da54e794
	github.com/lxn/win v0.0.0-20210218163916-a377121e959e
	github.com/romikb/amneziawg-windows v0.0.0-20240305122024-391d6c6b624f
	golang.org/x/crypto v0.18.0
	golang.org/x/sys v0.16.0
	golang.org/x/text v0.14.0
	golang.zx2c4.com/wireguard/windows v0.5.3
)

require (
	github.com/amnezia-vpn/amnezia-wg v0.1.8 // indirect
	github.com/tevino/abool/v2 v2.1.0 // indirect
	golang.org/x/mod v0.8.0 // indirect
	golang.org/x/net v0.16.0 // indirect
	golang.org/x/tools v0.6.0 // indirect
	golang.zx2c4.com/wintun v0.0.0-20230126152724-0fa3db229ce2 // indirect
)

replace (
	github.com/lxn/walk => golang.zx2c4.com/wireguard/windows v0.0.0-20210121140954-e7fc19d483bd
	github.com/lxn/win => golang.zx2c4.com/wireguard/windows v0.0.0-20210224134948-620c54ef6199
)
