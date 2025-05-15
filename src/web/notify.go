package web

import (
	"net/http"
	"strings"

	"IB1/config"
	"IB1/db"
	"IB1/util"
)

func notify(media string) error {
	if config.Cfg.Media.NotificationURL == "" { return nil }
	req, _ := http.NewRequest("POST", config.Cfg.Media.NotificationURL,
			strings.NewReader("New media file on " +
				config.Cfg.Web.Domain + " pending approval"))
	req.Header.Set("Title", "Pending media approval")
	secret, err := util.NewTextToken()
	if err != nil { return err }
	prefix := "http://" + config.Cfg.Web.Domain
	if config.Cfg.Web.BaseURL != "" {
		prefix = config.Cfg.Web.BaseURL
	}
	suffix := secret + "/" + media + ".png"
	thumbnail := prefix + "/media/thumbnail/" + suffix
	approve := prefix + "/approval/accept/" + suffix
	deny := prefix + "/approval/deny/" + suffix
	req.Header.Set("Attach", thumbnail)
	req.Header.Set("Actions",
		"http, Approve, " + approve + ", method=GET, clear=true; " +
		"http, Deny, " + deny + ", method=GET, clear=true; ")
	_, err = http.DefaultClient.Do(req)
	if err != nil { return err }
	return db.ApprovalBypass{}.Add(db.ApprovalBypass{
		Secret:	secret,
		Hash:	media,
	})
}
