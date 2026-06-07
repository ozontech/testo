package testoplugin

import (
	"context"
	"time"
)

// Overrides defines all builtin methods of T a plugin can override.
//
// Overrides work using middleware pattern - multiple overrides are stacked on top of each other,
// passing by a "next" function.
//
// There exists a certain hierarchy what method calls what underneath.
// For example, overriding Log will affect Error, Skip, Fatal and their printf equivalents.
type Overrides struct {
	// Priority defines global priority for these overrides.
	// Overrides with lower priority values are called first.
	Priority Priority

	Log      Override[FuncLog]
	Parallel Override[FuncParallel]
	TempDir  Override[FuncTempDir]
	Deadline Override[FuncDeadline]
	Context  Override[FuncContext]
	Cleanup  Override[FuncCleanup]

	// Setenv calls Cleanup to restore an environment variable.
	// On error, it calls Fatal.
	Setenv Override[FuncSetenv]

	// Chdir calls Cleanup to restore a current directory.
	// On error, it calls Fatal.
	Chdir Override[FuncChdir]

	// Error calls Log followed by Fail.
	Error Override[FuncError]

	// Skip calls Log followed by SkipNow.
	Skip    Override[FuncSkip]
	SkipNow Override[FuncSkipNow]
	Skipped Override[FuncSkipped]
	Fail    Override[FuncFail]
	FailNow Override[FuncFailNow]
	Failed  Override[FuncFailed]

	// Fatal calls Log followed by FailNow.
	Fatal Override[FuncFatal]
}

type (
	// FuncParallel describes [testing.T.Parallel] method.
	FuncParallel func()

	// FuncSetenv describes [testing.T.Setenv] method.
	FuncSetenv func(key, value string)

	// FuncTempDir describes [testing.T.TempDir] method.
	FuncTempDir func() string

	// FuncLog describes [testing.T.Log] method.
	FuncLog func(args ...any)

	// FuncDeadline describes [testing.T.Deadline] method.
	FuncDeadline func() (deadline time.Time, ok bool)

	// FuncContext describes [testing.T.Context] method.
	FuncContext func() context.Context

	// FuncChdir describes [testing.T.Chdir] method.
	FuncChdir func(dir string)

	// FuncError describes [testing.T.Error] method.
	FuncError func(args ...any)

	// FuncSkip describes [testing.T.Skip] method.
	FuncSkip func(args ...any)

	// FuncSkipNow describes [testing.T.SkipNow] method.
	FuncSkipNow func()

	// FuncSkipped describes [testing.T.Skipped] method.
	FuncSkipped func() bool

	// FuncFail describes [testing.T.Fail] method.
	FuncFail func()

	// FuncFailNow describes [testing.T.FailNow] method.
	FuncFailNow func()

	// FuncFailed describes [testing.T.Failed] method.
	FuncFailed func() bool

	// FuncFatal describes [testing.T.Fatal] method.
	FuncFatal func(args ...any)

	// FuncCleanup describes [testing.T.Cleanup] method.
	FuncCleanup func(f func())
)

// Override for the function.
//
// Nil value is valid and represents absence of override.
//
// Use [Override.Call] to safely call an override.
type Override[F any] func(f F) F

// Call returns an overridden f.
// If override is nil f is returned as is.
func (o Override[F]) Call(f F) F {
	if o == nil {
		return f
	}

	return o(f)
}
