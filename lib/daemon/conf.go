package main

type Config struct {
	Metadata	Metadata
}

type Metadata struct {
	AppID		string
	FriendlyName	string
	StateDirectory	string
}

type Exec struct {
	Target		string
	Arguments	[]string
}

type BusLaunch struct {
	Enable		bool
	Target		string
	Arguments	[]string
}

type ProcMgmt struct {
	Track		bool
}

type SysMgmt struct {
	Inhibit		bool
	GlobalShortcuts	bool
}

type NetworkOpts struct {
	Enable		bool
	Filter		bool
	FilterDest	[]string
}

type PrivacyOpts struct {
	X11			bool
	ClassicNotifications	bool
	Cameras			bool
	PipeWire		bool
	Input			bool
}

type AdvancedOpts struct {
	Zink			bool
	Qt5Compat		bool
	MprisName		string
	TrayWake		bool
	KDEStatus		bool
	FlatpakInfo		bool
}