# Change Log

## [0.0.9]

* Return error from createContainerURL if storage settings are not configured by @peterbom in https://github.com/Azure/aks-periscope/pull/156
* Remove old redundant deployment file. by @Tatsinnit in https://github.com/Azure/aks-periscope/pull/162
* Fix Node Logs Collector to use separate keys for each log file by @peterbom in https://github.com/Azure/aks-periscope/pull/166
* Support Kustomize for development and consuming tools by @peterbom in https://github.com/Azure/aks-periscope/pull/164
* Allow Periscope to run on Windows nodes by @peterbom in https://github.com/Azure/aks-periscope/pull/167
* Make it easier to run and debug tests locally by @peterbom in https://github.com/Azure/aks-periscope/pull/170
* Document the automated testing approach introduced earlier by @peterbom in https://github.com/Azure/aks-periscope/pull/172
* Add notes for differences in Windows behaviour by @peterbom in https://github.com/Azure/aks-periscope/pull/174
* Adding Microsoft SECURITY.MD by @microsoft-github-policy-service in https://github.com/Azure/aks-periscope/pull/175

Thanks to @peterbom, @rzhang628 

## [0.0.8]

* A few minor edits to README.md by @davefellows in #147
* Add pod disrupution budget information collector. by @Tatsinnit in #135
* Behaviour fix, Upload API fix. by @Tatsinnit in #138
* Use client-go and remove unnecessary kubectl. by @Tatsinnit in #136
* update v1beta1 apiextension to v1. by @Tatsinnit in #139
* Improve CI and add iptables and kubeletcmd test structure. by @Tatsinnit in #140
* Enable mechanism for container sas key to be passed. by @Tatsinnit in #143
* add systemlogs test. by @Tatsinnit in #149
* Temporary disabling non-compliant collectors from test cov. by @Tatsinnit in #144


Thanks to @sophsoph321, @peterbom, @davefellows, @rzhang628, @bcho, @SanyaKochhar, @johnsonshi for interactions, reviews and various enagements.