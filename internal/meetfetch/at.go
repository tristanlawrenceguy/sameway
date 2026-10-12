package meetfetch

// At is where Microsoft and Zoom answer. The zero value is their own
// addresses; an app is given its own through app.Options, so a test
// stands its fake in while others run beside it.
type At struct {
	MicrosoftLogin, Graph string
	ZoomOAuth, ZoomAPI    string
}

func (a At) microsoftLogin() string { return or(a.MicrosoftLogin, "https://login.microsoftonline.com") }
func (a At) graph() string          { return or(a.Graph, "https://graph.microsoft.com/v1.0") }
func (a At) zoomOAuth() string      { return or(a.ZoomOAuth, "https://zoom.us/oauth/token") }
func (a At) zoomAPI() string        { return or(a.ZoomAPI, "https://api.zoom.us/v2") }

func or(set, real string) string {
	if set == "" {
		return real
	}
	return set
}
