package main

type Config struct {
	Metadata	Metadata
	Exec		Exec
	BusActivation	BusLaunch
	Processes	ProcMgmt
	System		SysMgmt
	Network		NetworkOpts
	Privacy		PrivacyOpts
	Advanced	AdvancedOpts
	Path		string
	isModern	bool
}

type Metadata struct {
	AppID		string
	FriendlyName	string
	StateDirectory	string
}

type Exec struct {
	Target		string
	Arguments	[]string
	Overlay		bool
}

type BusLaunch struct {
	Enable		bool
	Target		string
	Arguments	[]string
}

type ProcMgmt struct {
	Track		bool
	Background	bool
}

type SysMgmt struct {
	InhibitSuspend	bool
	InhibitOnBehalf	bool

	// New-style device allow slice, possible values: (dgpu, input, camera, kvm)
	DeviceAllow	[]string

	// Deprecated: do not use
	GameMode	bool

	// Deprecated: do not use
	Virtualization	bool
}

type NetworkOpts struct {
	Enable		bool
	Filter		bool
	FilterDest	[]string
}

type PrivacyOpts struct {
	Lockdown		bool
	X11			bool
	ClassicNotifications	bool

	PipeWire		bool

	// Deprecated: do not use
	Cameras			bool
	Input			bool
}

type AdvancedOpts struct {
	Zink			bool
	Qt5Compat		bool
	MprisName		[]string
	TrayWake		bool
	KDEStatus		bool
	FlatpakInfo		bool
	Landlock		bool
	Debugging		bool
}

