# AWS Reaper

## About

The Reaper terminates forgotten AWS resources.

Reaper workflow:

1. Find all enabled resources, filter them, then fire events based on config
    - Event types include sending emails, posting events to Datadog (Statsd), tagging resources on AWS, stopping or killing resources, and more
    - Reaper uses the `Owner` tag on resources to notify the owner of the resource with options to Ignore (for a time), Whitelist, Terminate, or Stop each resource they own
2. Report statistics about the resources that were found
3. Terminate or Stop abandoned resources after a set amount of time

*Caution* This app is experimental because:
* doesn't have a lot of tests (Read: any), will add when SDK is updated
* it's maintained by an intern

## Building
* install glide per `https://github.com/Masterminds/glide`
* checkout repo
* install dependencies: `glide install`
* build binary: `go build main.go`
* run binary: `./reaper -config config/default.toml`
* for command line options: `./reaper -help`

## Command Line Flags
* config: required flag, the path to the Reaper config file. `string` (no default value)
* dryrun: run Reaper in dryrun (no-op) mode. Events will not be triggered. `boolean` (default: true)
* interactive: run Reaper in Interactive mode, disabling all other Events, and prompting for input on Reapable discovery. `boolean` (default: false)
* load: load Reaper state from the StateFile (overrides tags with state in AWS) `boolean` (default: false)
* withoutCloudformationResources: skip checking for Cloudformation Resource dependencies (throttled by AWS, so it takes ages). `boolean` (default: false)

## Creating a configuration file
Reaper configuration files should be in toml format. See `config/default.toml` for an example config.

* Top level options
    - StateFile: the full filepath of the file that resource states (in custom format) are saved and loaded to / from. `string`
    - LogFile: the full filepath of the file that logs are written to. `string`
    - WhitelistTag: a string that will be used to tag resources that have been whitelisted. Defaults to `REAPER_SPARE_ME`. (string)
    - DefaultOwner: all unowned resources will be assigned to this owner. Can be an email address, or can be a username if DefaultEmailHost is specified. `string`
    - DefaultEmailHost: resources that do not have a complete email address as their owner will have this appended. Should be of the form "domain.tld". Works with DefaultOwner in the following way: `DefaultOwner`@`DefaultEmailHost`. `string`
    - EventTag: a tag that is added to all events that support tagging. Should be of the form `key1:value1,key2:value2`. `string`
* HTTP options (under `[HTTP]`)
    - TokenSecret: the secret key used to secure web requests. `string`
    - ApiURL: used to generate URLs for Reaper's HTTP API. Should be of the form `protocol://host:port`. `string`
    - Listen: where the HTTP server will listen for requests. Should be of the form `host:port`. `string`
    - Token: TODO
    - Action: TODO
* Logging (under `[Logging]`)
    - Extras: enables or disables extra logging, such as dry run notifications for EventReporters not triggering. `boolean`
* States (under `[States]`)
    - Interval: the interval between Reaper's scans for resources. The time format must be a duration parsable by Go's time.ParseDuration. See: http://godoc.org/time#ParseDuration. Example: `1h`. `string`
    - FirstStateDuration: the length of the first state assigned to resources that match filters. The time format must be a duration parsable by Go's time.ParseDuration. See: http://godoc.org/time#ParseDuration. Example: `1h`. `string`
    - SecondStateDuration: the length of the second state assigned to resources that match filters. The time format must be a duration parsable by Go's time.ParseDuration. See: http://godoc.org/time#ParseDuration. Example: `1h`. `string`
    - ThirdStateDuration: the length of the third state assigned to resources that match filters. After the Third state elapses, resources move to a permanent final state. The time format must be a duration parsable by Go's time.ParseDuration. See: http://godoc.org/time#ParseDuration. Example: `1h`. `string`
* Events (under `[Events]`)
    - Datadog (`[Events.Datadog]`)
        + Enabled: enables or disables the Datadog EventReporter. Note: Datadog statistics and Event depend on this. `boolean`
        + Triggers: states for which Datadog will trigger Reapable Events. Can be any/all/none of `first`, `second`, `third`, `final`, or `ignore`. `[]string`
    - Tagger (`[Events.Tagger]`)
        + Enabled: enables or disables the Tagger EventReporter. `boolean`
        + Triggers: states for which Tagger will trigger Reapable Events. Can be any/all/none of `first`, `second`, `third`, `final`, or `ignore`. `[]string`
    - Reaper (`[Events.Reaper]`)
        + Enabled: enables or disables the Reaper EventReporter. `boolean`
        + Triggers: states for which Reaper will trigger Reapable Events. Can be any/all/none of `first`, `second`, `third`, `final`, or `ignore`. `[]string`
        + Mode: when the Reaper EventReporter is triggered on a Reapable Event, it will `Stop` or `Terminate` Reapables per this flag. Note: modes must be capitalized. `string`
    - Email (`[Events.Email]`)
        + Enabled: enables or disables the Email EventReporter. `boolean`
        + Triggers: states for which Email will trigger Reapable Events. Can be any/all/none of `first`, `second`, `third`, `final`, or `ignore`. `[]string`
        + Host: the mailserver Reaper will use
        + AuthType: the type of authentication used by the mailserver. Should be one of `none`, `md5` or `plain`. `string`
        + Port: the port used by the mailserver. `int`
        + Username: the username to use for the mailserver. `string`
        + Password: the password to use for the nmailserver. `string`
        + From: the address that Reaper will send mail from, must be parsable by Go's mail.ParseAddress. See: http://godoc.org/net/mail#ParseAddress. `string`
* All Supported AWS Resource types have these properties
    - Enabled: enables or disables reporting of this resource type. Note: resources will still be queried for as they inform Reaper about the dependencies of other resources. `boolean`
    - FilterGroups (under `[ResourceType.FilterGroups]`): FilterGroups are sets of filters that can be applied to resources. In order for a resource to match a FilterGroup, it must match _all_ filters in the FilterGroup. If an resource matches _any_ FilterGroup, it has satisfied Reaper's filters. `[]FilterGroup`
        + Example FilterGroup:
            ```
            [ResourceType.FilterGroups.Example]
                    [ResourceType.FilterGroups.Example.Filter1]
                        function = "IsDependency"
                        arguments = ["false"]
                    [ResourceType.FilterGroups.Example.Filter2]
                        function = "Running"
                        arguments = ["true"]
            ```

        + In this example, we see a FilterGroup named "Example" that has two Filters, Filter1 and Filter2.
        + A FilterGroup is a `[]Filter`, and a Filter has two components, a `function` and `arguments`. The `function` is the name of the filtering function for the associated resource type (`string`), and `arguments` is a slice of arguments to that function (`[]string`).
* Currently supported AWS Resource types:
    - SecurityGroups (under `[SecurityGroups]`)
    - Cloudformations (under `[Cloudformations]`)
    - AutoScalingGroups (under `[AutoScalingGroups]`)
    - Instances (under `[Instances]`)
    - Volumes (under `[Volumes]`)
