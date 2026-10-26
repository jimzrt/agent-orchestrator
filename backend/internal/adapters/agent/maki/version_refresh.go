package maki

// SupportsTerminalVersionRefresh leaves unattended upgrades off: restore uses
// `maki --continue` rather than an exact native conversation id.
func (*Plugin) SupportsTerminalVersionRefresh() bool { return false }
