# rungroup :rainbow:

[![GoDoc](https://pkg.go.dev/badge/github.com/bharat-rajani/rungroup)](https://godoc.org/github.com/bharat-rajani/rungroup)
![Build](https://github.com/bharat-rajani/rungroup/actions/workflows/push.yml/badge.svg)
[![Codecov](https://codecov.io/gh/bharat-rajani/rungroup/branch/main/graph/badge.svg?token=K0JRWEWYWX)](https://codecov.io/gh/bharat-rajani/rungroup)
[![GoVersion](https://img.shields.io/github/go-mod/go-version/bharat-rajani/rungroup)](https://github.com/bharat-rajani/rungroup/blob/main/go.mod)
[![Go Report Card](https://img.shields.io/badge/go%20report-A+-brightgreen.svg)](https://goreportcard.com/report/github.com/bharat-rajani/rungroup)
[![MIT licensed](https://img.shields.io/github/license/bharat-rajani/rungroup)](https://github.com/bharat-rajani/rungroup/blob/main/LICENSE)

### Goroutines lifecycle manager :bug: :butterfly: :coffin:

RunGroup is a Go package that is designed to effectively manage concurrent tasks within a group, allowing you to track and store their results using a result map. It includes features for setting up contexts, running interrupting concurrent tasks, and retrieving results.

Table of contents
=================

- [Installation](#installation-floppy_disk)
- [Usage](#usage)
- [API Reference](#api-reference)
- [Examples](#examples)
- [Contributing](#contributing)
- [License](#license)


Installation:floppy_disk:
=================

```shell
go get -u github.com/bharat-rajani/rungroup/v2   
```

Usage
=================

RunGroup is designed to help you manage concurrent tasks and their errors efficiently. Here's how you can use it:


## Setting up a Group with Context and Result Map
You can create a new concurrent task group with a specific context and result map using the provided functions:

```go
group, ctx := concurrent.WithContextResultMap[TaskIdentifierType, TaskOutputType](parentContext, resultMap)
```

OR

```go
group, ctx := concurrent.WithContext[TaskIdentifierType, TaskOutputType](parentContext)
```

The first option (WithContextResultMap) initializes the group with a specific result map for storing task results, while the second option (WithContext) sets up the group without a result map.

## Running Concurrent Tasks
To run concurrent tasks within the group, use the `GoWithFunc` method. You can associate each task with a unique identifier for tracking purposes:

```go
ctx := context.Background()
group.GoWithFunc(func() (V, error) {
    // Your task logic here
    return result, err
},ctx, interrupter, taskID)
```


- func() (V, error): The function to execute concurrently.
- interrupter: A boolean flag indicating whether the task can be interrupted upon error.
- taskID: A unique identifier associated with the task.


## Retrieving Task Results
You can retrieve task results using the GetResultByID method, which takes the task's unique identifier as a parameter:

```go
result, err, found := group.GetResultByID(taskID)
```

- result: The result value.
- err: An error, if any.
- found: A boolean indicating whether the result was found.

API Reference
=================

GoDoc
For detailed information about available functions and types, refer to the GoDoc documentation.

Examples
=================

Explore the [examples](examples/) directory for usage examples and sample code.

### Contributing
Contributions to this project are welcome. Please open an issue or submit a pull request for any improvements or bug fixes.

### License
This project is licensed under the MIT License. See the LICENSE file for details.

> Rungroup is inspired by [errorgroup]( https://github.com/golang/sync/blob/master/errgroup/errgroup.go).
