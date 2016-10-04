package aws

import (
	"net/url"
	"time"

	"github.com/mozilla-services/reaper/reapable"
	log "github.com/mozilla-services/reaper/reaperlog"
	"github.com/mozilla-services/reaper/token"
)

// MakeScheduleLink creates a tokenized link scheduling a resource
func makeScheduleLink(region reapable.Region, id reapable.ID, tokenSecret, apiURL, scaleDownSchedule, scaleUpSchedule string) (string, error) {
	sched, err := token.Tokenize(tokenSecret,
		token.NewScheduleJob(region.String(), id.String(), scaleDownSchedule, scaleUpSchedule))

	if err != nil {
		return "", err
	}

	return makeURL(apiURL, "schedule", sched), nil
}

// MakeTerminateLink creates a tokenized link scheduling a resource
func makeTerminateLink(region reapable.Region, id reapable.ID, tokenSecret, apiURL string) (string, error) {
	term, err := token.Tokenize(tokenSecret,
		token.NewTerminateJob(region.String(), id.String()))

	if err != nil {
		return "", err
	}

	return makeURL(apiURL, "terminate", term), nil
}

// MakeIgnoreLink creates a tokenized link scheduling a resource
func makeIgnoreLink(region reapable.Region, id reapable.ID, tokenSecret, apiURL string,
	duration time.Duration) (string, error) {
	delay, err := token.Tokenize(tokenSecret,
		token.NewDelayJob(region.String(), id.String(),
			duration))

	if err != nil {
		return "", err
	}

	action := "delay_" + duration.String()
	return makeURL(apiURL, action, delay), nil

}

// MakeWhitelistLink creates a tokenized link scheduling a resource
func makeWhitelistLink(region reapable.Region, id reapable.ID, tokenSecret, apiURL string) (string, error) {
	whitelist, err := token.Tokenize(tokenSecret,
		token.NewWhitelistJob(region.String(), id.String()))
	if err != nil {
		log.Error("Error creating whitelist link: ", err)
		return "", err
	}

	return makeURL(apiURL, "whitelist", whitelist), nil
}

// MakeStopLink creates a tokenized link scheduling a resource
func makeStopLink(region reapable.Region, id reapable.ID, tokenSecret, apiURL string) (string, error) {
	stop, err := token.Tokenize(tokenSecret,
		token.NewStopJob(region.String(), id.String()))
	if err != nil {
		log.Error("Error creating ScaleToZero link: ", err)
		return "", err
	}

	return makeURL(apiURL, "stop", stop), nil
}

// MakeForceStopLink creates a tokenized link scheduling a resource
func makeForceStopLink(region reapable.Region, id reapable.ID, tokenSecret, apiURL string) (string, error) {
	stop, err := token.Tokenize(tokenSecret,
		token.NewForceStopJob(region.String(), id.String()))
	if err != nil {
		log.Error("Error creating ScaleToZero link: ", err)
		return "", err
	}

	return makeURL(apiURL, "stop", stop), nil
}

func makeURL(host, action, token string) string {
	if host == "" {
		log.Error("makeURL: host is empty")
	}

	action = url.QueryEscape(action)
	token = url.QueryEscape(token)

	vals := url.Values{}
	vals.Add(config.HTTP.Action, action)
	vals.Add(config.HTTP.Token, token)

	if host[len(host)-1:] == "/" {
		return host + "?" + vals.Encode()
	}
	return host + "/?" + vals.Encode()
}
