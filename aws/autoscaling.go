package aws

import (
	"bytes"
	"fmt"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"

	"github.com/mozilla-services/reaper/filters"
	"github.com/mozilla-services/reaper/reapable"
	log "github.com/mozilla-services/reaper/reaperlog"
	"github.com/mozilla-services/reaper/state"
)

// AutoScalingGroup is a Reapable, Filterable
// embeds AWS API's autoscaling.Group
type AutoScalingGroup struct {
	Resource
	autoscaling.Group

	// autoscaling.Instance exposes minimal info
	Instances []reapable.ID
}

// NewAutoScalingGroup creates an AutoScalingGroup from the AWS API's autoscaling.Group
func NewAutoScalingGroup(region string, asg *autoscaling.Group) *AutoScalingGroup {
	a := AutoScalingGroup{
		Resource: Resource{
			region: reapable.Region(region),
			id:     reapable.ID(*asg.AutoScalingGroupName),
			Name:   *asg.AutoScalingGroupName,
			Tags:   make(map[string]string),
		},
		Group: *asg,
	}

	for _, instance := range asg.Instances {
		a.Instances = append(a.Instances, reapable.ID(*instance.InstanceId))
	}

	for _, tag := range asg.Tags {
		a.Resource.Tags[*tag.Key] = *tag.Value
	}

	if a.Tagged("aws:cloudformation:stack-name") {
		a.Dependency = true
		a.IsInCloudformation = true
	}

	if a.Tagged(reaperTag) {
		// restore previously tagged state
		a.reaperState = state.NewStateWithTag(a.Tag(reaperTag))
	} else {
		// initial state
		a.reaperState = state.NewState()
	}

	return &a
}

// ReapableEventText is part of the events.Reapable interface
func (a *AutoScalingGroup) ReapableEventText() (*bytes.Buffer, error) {
	return reapableEventText(a, reapableASGEventText)
}

// ReapableEventTextShort is part of the events.Reapable interface
func (a *AutoScalingGroup) ReapableEventTextShort() (*bytes.Buffer, error) {
	return reapableEventText(a, reapableASGEventTextShort)
}

// ReapableEventEmail is part of the events.Reapable interface
func (a *AutoScalingGroup) ReapableEventEmail() (owner mail.Address, subject string, body *bytes.Buffer, err error) {
	// if unowned, return unowned error
	if !a.Owned() {
		err = reapable.UnownedError{ErrorText: fmt.Sprintf("%s does not have an owner tag", a.ReapableDescriptionShort())}
		return
	}

	subject = fmt.Sprintf("AWS Resource %s is going to be Reaped!", a.ReapableDescriptionTiny())
	owner = *a.Owner()
	body, err = reapableEventHTML(a, reapableASGEventHTML)
	return
}

// ReapableEventEmailShort is part of the events.Reapable interface
func (a *AutoScalingGroup) ReapableEventEmailShort() (owner mail.Address, body *bytes.Buffer, err error) {
	// if unowned, return unowned error
	if !a.Owned() {
		err = reapable.UnownedError{ErrorText: fmt.Sprintf("%s does not have an owner tag", a.ReapableDescriptionShort())}
		return
	}
	owner = *a.Owner()
	body, err = reapableEventHTML(a, reapableASGEventHTMLShort)
	return
}

type autoScalingGroupEventData struct {
	Config           *Config
	AutoScalingGroup *AutoScalingGroup
	TerminateLink    string
	StopLink         string
	WhitelistLink    string
	IgnoreLink1      string
	IgnoreLink3      string
	IgnoreLink7      string
}

func (a *AutoScalingGroup) getTemplateData() (interface{}, error) {
	ignore1, err := makeIgnoreLink(a.Region(), a.ID(), config.HTTP.TokenSecret, config.HTTP.APIURL, time.Duration(1*24*time.Hour))
	if err != nil {
		return nil, err
	}
	ignore3, err := makeIgnoreLink(a.Region(), a.ID(), config.HTTP.TokenSecret, config.HTTP.APIURL, time.Duration(3*24*time.Hour))
	if err != nil {
		return nil, err
	}
	ignore7, err := makeIgnoreLink(a.Region(), a.ID(), config.HTTP.TokenSecret, config.HTTP.APIURL, time.Duration(7*24*time.Hour))
	if err != nil {
		return nil, err
	}
	terminate, err := makeTerminateLink(a.Region(), a.ID(), config.HTTP.TokenSecret, config.HTTP.APIURL)
	if err != nil {
		return nil, err
	}
	stop, err := makeStopLink(a.Region(), a.ID(), config.HTTP.TokenSecret, config.HTTP.APIURL)
	if err != nil {
		return nil, err
	}
	whitelist, err := makeWhitelistLink(a.Region(), a.ID(), config.HTTP.TokenSecret, config.HTTP.APIURL)
	if err != nil {
		return nil, err
	}

	return &autoScalingGroupEventData{
		Config:           config,
		AutoScalingGroup: a,
		TerminateLink:    terminate,
		StopLink:         stop,
		WhitelistLink:    whitelist,
		IgnoreLink1:      ignore1,
		IgnoreLink3:      ignore3,
		IgnoreLink7:      ignore7,
	}, nil
}

const reapableASGEventHTML = `
<html>
<body>
	<p>AutoScalingGroup <a href="{{ .AutoScalingGroup.AWSConsoleURL }}">{{ if .AutoScalingGroup.Name }}"{{.AutoScalingGroup.Name}}" {{ end }} in {{.AutoScalingGroup.Region}}</a> is scheduled to be terminated.</p>

	<p>
		You can ignore this message and your AutoScalingGroup will advance to the next state after <strong>{{.AutoScalingGroup.ReaperState.Until}}</strong>. If you do not take action it will be terminated!
	</p>

	<p>
		You may also choose to:
		<ul>
			<li><a href="{{ .TerminateLink }}">Terminate it now</a></li>
			<li><a href="{{ .StopLink }}">Scale it to 0</a></li>
			<li><a href="{{ .IgnoreLink1 }}">Ignore it for 1 more day</a></li>
			<li><a href="{{ .IgnoreLink3 }}">Ignore it for 3 more days</a></li>
			<li><a href="{{ .IgnoreLink7}}">Ignore it for 7 more days</a></li>
		</ul>
	</p>

	<p>
		If you want the Reaper to ignore this AutoScalingGroup tag it with {{ .Config.WhitelistTag }} with any value, or click <a href="{{ .WhitelistLink }}">here</a>.
	</p>
</body>
</html>
`

const reapableASGEventHTMLShort = `
<html>
<body>
	<p>AutoScalingGroup <a href="{{ .AutoScalingGroup.AWSConsoleURL }}">{{ if .AutoScalingGroup.Name }}"{{.AutoScalingGroup.Name}}" {{ end }}</a> in {{.AutoScalingGroup.Region}}</a> is scheduled to be terminated after <strong>{{.AutoScalingGroup.ReaperState.Until}}</strong>.
		<br />
		<a href="{{ .TerminateLink }}">Terminate</a>,
		<a href="{{ .StopLink }}">Stop</a>,
		<a href="{{ .IgnoreLink1 }}">Ignore it for 1 more day</a>,
		<a href="{{ .IgnoreLink3 }}">3 days</a>,
		<a href="{{ .IgnoreLink7}}"> 7 days</a>,
		<a href="{{ .WhitelistLink }}">Whitelist</a> it.
	</p>
</body>
</html>
`

const reapableASGEventTextShort = `%%%
AutoScalingGroup [{{.AutoScalingGroup.ID}}]({{.AutoScalingGroup.AWSConsoleURL}}) in region: [{{.AutoScalingGroup.Region}}](https://{{.AutoScalingGroup.Region}}.console.aws.amazon.com/ec2/v2/home?region={{.AutoScalingGroup.Region}}).{{if .AutoScalingGroup.Owned}} Owned by {{.AutoScalingGroup.Owner}}.\n{{end}}
[Whitelist]({{ .WhitelistLink }}), [Scale to 0]({{ .StopLink }}), or [Terminate]({{ .TerminateLink }}) this AutoScalingGroup.
%%%`

const reapableASGEventText = `%%%
Reaper has discovered an AutoScalingGroup qualified as reapable: [{{.AutoScalingGroup.ID}}]({{.AutoScalingGroup.AWSConsoleURL}}) in region: [{{.AutoScalingGroup.Region}}](https://{{.AutoScalingGroup.Region}}.console.aws.amazon.com/ec2/v2/home?region={{.AutoScalingGroup.Region}}).\n
{{if .AutoScalingGroup.Owned}}Owned by {{.AutoScalingGroup.Owner}}.\n{{end}}
{{ if .AutoScalingGroup.AWSConsoleURL}}{{.AutoScalingGroup.AWSConsoleURL}}\n{{end}}
[AWS Console URL]({{.AutoScalingGroup.AWSConsoleURL}})\n
[Whitelist]({{ .WhitelistLink }}) this AutoScalingGroup.
[Scale to 0]({{ .StopLink }}) this AutoScalingGroup.
[Terminate]({{ .TerminateLink }}) this AutoScalingGroup.
%%%`

func (a *AutoScalingGroup) sizeGreaterThanOrEqualTo(size int64) bool {
	if a.DesiredCapacity != nil {
		return *a.DesiredCapacity >= size
	}
	return false
}

func (a *AutoScalingGroup) sizeLessThanOrEqualTo(size int64) bool {
	if a.DesiredCapacity != nil {
		return *a.DesiredCapacity <= size
	}
	return false
}

func (a *AutoScalingGroup) sizeEqualTo(size int64) bool {
	if a.DesiredCapacity != nil {
		return *a.DesiredCapacity == size
	}
	return false
}

func (a *AutoScalingGroup) sizeLessThan(size int64) bool {
	if a.DesiredCapacity != nil {
		return *a.DesiredCapacity < size
	}
	return false
}

func (a *AutoScalingGroup) sizeGreaterThan(size int64) bool {
	if a.DesiredCapacity != nil {
		return *a.DesiredCapacity > size
	}
	return false
}

// Save is part of reapable.Saveable, which embedded in reapable.Reapable
func (a *AutoScalingGroup) Save(s *state.State) (bool, error) {
	return tagAutoScalingGroup(a.Region(), a.ID(), reaperTag, a.reaperState.String())
}

// Unsave is part of reapable.Saveable, which embedded in reapable.Reapable
func (a *AutoScalingGroup) Unsave() (bool, error) {
	log.Info("Unsaving %s", a.ReapableDescriptionTiny())
	return untagAutoScalingGroup(a.Region(), a.ID(), reaperTag)
}

func untagAutoScalingGroup(region reapable.Region, id reapable.ID, key string) (bool, error) {
	api := autoscaling.New(session.New(&aws.Config{Region: aws.String(string(region))}))
	deletereq := &autoscaling.DeleteTagsInput{
		Tags: []*autoscaling.Tag{
			&autoscaling.Tag{
				ResourceId:   aws.String(string(id)),
				ResourceType: aws.String("auto-scaling-group"),
				Key:          aws.String(key),
			},
		},
	}

	_, err := api.DeleteTags(deletereq)
	if err != nil {
		return false, err
	}

	return true, nil
}

func tagAutoScalingGroup(region reapable.Region, id reapable.ID, key, value string) (bool, error) {
	log.Info("Tagging AutoScalingGroup %s in %s with %s:%s", region.String(), id.String(), key, value)
	api := autoscaling.New(session.New(&aws.Config{Region: aws.String(region.String())}))
	createreq := &autoscaling.CreateOrUpdateTagsInput{
		Tags: []*autoscaling.Tag{
			&autoscaling.Tag{
				ResourceId:        aws.String(string(id)),
				ResourceType:      aws.String("auto-scaling-group"),
				PropagateAtLaunch: aws.Bool(false),
				Key:               aws.String(key),
				Value:             aws.String(value),
			},
		},
	}

	_, err := api.CreateOrUpdateTags(createreq)
	if err != nil {
		return false, err
	}

	return true, nil
}

// Filter is part of the filter.Filterable interface
func (a *AutoScalingGroup) Filter(filter filters.Filter) bool {
	matched := false
	// map function names to function calls
	switch filter.Function {
	case "SizeGreaterThan":
		if i, err := filter.Int64Value(0); err == nil && a.sizeGreaterThan(i) {
			matched = true
		}
	case "SizeLessThan":
		if i, err := filter.Int64Value(0); err == nil && a.sizeLessThan(i) {
			matched = true
		}
	case "SizeEqualTo":
		if i, err := filter.Int64Value(0); err == nil && a.sizeEqualTo(i) {
			matched = true
		}
	case "SizeLessThanOrEqualTo":
		if i, err := filter.Int64Value(0); err == nil && a.sizeLessThanOrEqualTo(i) {
			matched = true
		}
	case "SizeGreaterThanOrEqualTo":
		if i, err := filter.Int64Value(0); err == nil && a.sizeGreaterThanOrEqualTo(i) {
			matched = true
		}
	case "CreatedTimeInTheLast":
		d, err := time.ParseDuration(filter.Arguments[0])
		if err == nil && a.CreatedTime != nil && time.Since(*a.CreatedTime) < d {
			matched = true
		}
	case "CreatedTimeNotInTheLast":
		d, err := time.ParseDuration(filter.Arguments[0])
		if err == nil && a.CreatedTime != nil && time.Since(*a.CreatedTime) > d {
			matched = true
		}
	case "InCloudformation":
		if b, err := filter.BoolValue(0); err == nil && a.IsInCloudformation == b {
			matched = true
		}
	case "Region":
		for _, region := range filter.Arguments {
			if a.Region() == reapable.Region(region) {
				matched = true
			}
		}
	case "NotRegion":
		// was this resource's region one of those in the NOT list
		regionSpecified := false
		for _, region := range filter.Arguments {
			if a.Region() == reapable.Region(region) {
				regionSpecified = true
			}
		}
		if !regionSpecified {
			matched = true
		}
	case "Tagged":
		if a.Tagged(filter.Arguments[0]) {
			matched = true
		}
	case "NotTagged":
		if !a.Tagged(filter.Arguments[0]) {
			matched = true
		}
	case "TagNotEqual":
		if a.Tag(filter.Arguments[0]) != filter.Arguments[1] {
			matched = true
		}
	case "ReaperState":
		if a.reaperState.State.String() == filter.Arguments[0] {
			matched = true
		}
	case "NotReaperState":
		if a.reaperState.State.String() != filter.Arguments[0] {
			matched = true
		}
	case "Named":
		if a.Name == filter.Arguments[0] {
			matched = true
		}
	case "NotNamed":
		if a.Name != filter.Arguments[0] {
			matched = true
		}
	case "IsDependency":
		if b, err := filter.BoolValue(0); err == nil && a.Dependency == b {
			matched = true
		}
	case "NameContains":
		if strings.Contains(a.Name, filter.Arguments[0]) {
			matched = true
		}
	case "NotNameContains":
		if !strings.Contains(a.Name, filter.Arguments[0]) {
			matched = true
		}
	default:
		log.Error(fmt.Sprintf("No function %s could be found for filtering AutoScalingGroups.", filter.Function))
	}
	return matched
}

// AWSConsoleURL returns the url that can be used to access the resource on the AWS Console
func (a *AutoScalingGroup) AWSConsoleURL() *url.URL {
	url, err := url.Parse(fmt.Sprintf("https://%s.console.aws.amazon.com/ec2/autoscaling/home?region=%s#AutoScalingGroups:id=%s;view=details",
		a.Region().String(), a.Region().String(), url.QueryEscape(a.ID().String())))
	if err != nil {
		log.Error("Error generating AWSConsoleURL. ", err)
	}
	return url
}

func (a *AutoScalingGroup) scaleToSize(size int64, minSize int64) (bool, error) {
	log.Info("Scaling AutoScalingGroup %s to size %d.", a.ReapableDescriptionTiny(), size)
	as := autoscaling.New(session.New(&aws.Config{Region: aws.String(a.Region().String())}))
	input := &autoscaling.UpdateAutoScalingGroupInput{
		AutoScalingGroupName: aws.String(a.ID().String()),
		DesiredCapacity:      &size,
		MinSize:              &minSize,
	}

	_, err := as.UpdateAutoScalingGroup(input)
	if err != nil {
		log.Error("could not update AutoScalingGroup ", a.ReapableDescriptionTiny())
		return false, err
	}
	return true, nil
}

// Terminate is a method of reapable.Terminable, which is embedded in reapable.Reapable
func (a *AutoScalingGroup) Terminate() (bool, error) {
	log.Info("Terminating AutoScalingGroup %s", a.ReapableDescriptionTiny())
	as := autoscaling.New(session.New(&aws.Config{Region: aws.String(a.Region().String())}))
	input := &autoscaling.DeleteAutoScalingGroupInput{
		AutoScalingGroupName: aws.String(a.ID().String()),
	}
	_, err := as.DeleteAutoScalingGroup(input)
	if err != nil {
		log.Error("could not delete AutoScalingGroup ", a.ReapableDescriptionTiny())
		return false, err
	}
	return true, nil
}

// Whitelist is a method of reapable.Whitelistable, which is embedded in reapable.Reapable
func (a *AutoScalingGroup) Whitelist() (bool, error) {
	log.Info("Whitelisting AutoScalingGroup %s", a.ReapableDescriptionTiny())
	api := autoscaling.New(session.New(&aws.Config{Region: aws.String(a.Region().String())}))
	createreq := &autoscaling.CreateOrUpdateTagsInput{
		Tags: []*autoscaling.Tag{
			&autoscaling.Tag{
				ResourceId:        aws.String(a.ID().String()),
				ResourceType:      aws.String("auto-scaling-group"),
				PropagateAtLaunch: aws.Bool(false),
				Key:               aws.String(config.WhitelistTag),
				Value:             aws.String("true"),
			},
		},
	}
	_, err := api.CreateOrUpdateTags(createreq)
	return err == nil, err
}

// Stop is a method of reapable.Stoppable, which is embedded in reapable.Reapable
// Stop scales ASGs to 0
func (a *AutoScalingGroup) Stop() (bool, error) {
	// use existing min size
	return a.scaleToSize(0, 0)
}
