package main

// Command is request commands as defined in RFC 1928 section 4.
type Command uint8

// SOCKS request commands as defined in RFC 1928 section 4.
const (
	CmdConnect      Command = 0x01
	CmdBind         Command = 0x02
	CmdUDPAssociate Command = 0x03
	CmdICMP         Command = 0x04
	CmdGatewaySate  Command = 0x05
	CmdTraceroute   Command = 0x09
)

func (c Command) String() string {
	switch c {
	case CmdConnect:
		return "CONNECT"
	case CmdBind:
		return "BIND"
	case CmdUDPAssociate:
		return "UDP ASSOCIATE"
	case CmdICMP:
		return "ICMP"
	default:
		return "UNDEFINED"
	}
}
