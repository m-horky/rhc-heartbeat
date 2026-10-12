package rhsm

// dtoConfig contains one partial RHSM configuration layer.
type dtoConfig struct {
	Server dtoServer
	RHSM   dtoRHSM
}

// dtoServer contains optional subscription-server and proxy settings.
type dtoServer struct {
	Hostname      *string
	Prefix        *string // API path prefix
	Port          *string
	Insecure      *bool
	ProxyHostname *string
	ProxyScheme   *string
	ProxyPort     *string
	ProxyUser     *string
	ProxyPassword *string
	NoProxy       *string
}

// dtoRHSM contains optional CA settings before INI interpolation.
type dtoRHSM struct {
	CACertDir  *string
	RepoCACert *string
}

// update validates and merges supplied RHSM server leaves into the receiver.
func (partial *dtoConfig) update(next *dtoConfig) error {
	// TODO Should we prefer a new object over mutation?
	if next == nil {
		return nil
	}

	if err := partial.Server.update(&next.Server); err != nil {
		return err
	}

	if err := partial.RHSM.update(&next.RHSM); err != nil {
		return err
	}

	return nil
}

// update merges subscription-server and proxy preferences.
func (partial *dtoServer) update(next *dtoServer) error {
	if next == nil {
		return nil
	}

	if next.Hostname != nil {
		partial.Hostname = new(*next.Hostname)
	}

	if next.Prefix != nil {
		partial.Prefix = new(*next.Prefix)
	}

	if next.Port != nil {
		partial.Port = new(*next.Port)
	}

	if next.Insecure != nil {
		partial.Insecure = next.Insecure
	}

	if next.ProxyHostname != nil {
		partial.ProxyHostname = new(*next.ProxyHostname)
	}

	if next.ProxyScheme != nil {
		partial.ProxyScheme = new(*next.ProxyScheme)
	}

	if next.ProxyPort != nil {
		partial.ProxyPort = new(*next.ProxyPort)
	}

	if next.ProxyUser != nil {
		partial.ProxyUser = new(*next.ProxyUser)
	}

	if next.ProxyPassword != nil {
		partial.ProxyPassword = new(*next.ProxyPassword)
	}

	if next.NoProxy != nil {
		partial.NoProxy = new(*next.NoProxy)
	}

	return nil
}

// update merges certificate authority preferences.
func (partial *dtoRHSM) update(next *dtoRHSM) error {
	if next == nil {
		return nil
	}

	if next.CACertDir != nil {
		partial.CACertDir = new(*next.CACertDir)
	}

	if next.RepoCACert != nil {
		partial.RepoCACert = new(*next.RepoCACert)
	}

	return nil
}
