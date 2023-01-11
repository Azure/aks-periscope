## Local Development

Ginkgo is used to test the package. to test locally, install ginkgo if required. change into this directory (`pkg/collector/gadgets`)

```shell
# the gadget needs to run with root permission, so we build it first, which outputs gadgets.test binary
ginko build


sudo ./gadgets.test

```
