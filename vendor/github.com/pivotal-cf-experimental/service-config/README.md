# service-config

[![Build Status](https://travis-ci.org/pivotal-cf-experimental/service-config.svg)](https://travis-ci.org/pivotal-cf-experimental/service-config)

Tiny Go library for flexible config file loading.

## Why?

To allow a service to be configured in multiple different ways:
- Enable usage by both humans and machines
- Enable usage both locally and remote
- Optionally satisfy [12 factor](http://12factor.net/) constraints

## Features

Loading mechanisms (in order of precedence):
- Config as flag json string: `-config={ "key": "value" }`
- Config as flag path to json file: `-configPath=/path/to/config.json`
- The `-h` flag will print the above flags, as well as any defaults specified using the `AddDefaults` method
- Config as environment variable json string: `CONFIG={ "key": "value" }`
- Config as environment variable path to json file: `CONFIG_PATH=/path/to/config.json`

## Usage

See the [example service](examples/test_service.go) for usage.

## Dependencies

service-config depends on [yaml](https://github.com/go-yaml/yaml) for json & yaml parsing and
on [mergo](https://github.com/imdario/mergo) merging defaults.

```
$ go get -u github.com/go-yaml/yaml
$ go get -u github.com/imdario/mergo
```

## Limitations

A non-empty default value can not be overridden by an empty provided value.

To allow empty values in the config properties (e.g. `-config={ "password": "" }`),
   the default value for that property must also be empty.

For example, the following default config would allow a blank password to be provided:
```
serviceConfig.AddDefaults(Config{
   password: "",
})
```

While this example would not allow the provided password to be blank:
```
serviceConfig.AddDefaults(Config{
   password: "password",
})
```

## License

Copyright 2015 Pivotal Software, Inc

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
