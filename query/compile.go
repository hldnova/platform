package query

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"time"

	"github.com/influxdata/platform/query/interpreter"
	"github.com/influxdata/platform/query/parser"
	"github.com/influxdata/platform/query/semantic"
	"github.com/influxdata/platform/query/values"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
)

const (
	TableParameter  = "table"
	tableKindKey    = "kind"
	tableParentsKey = "parents"
	nowOption       = "now"
	//tableSpecKey    = "spec"
)

type Option func(*options)

func Verbose(v bool) Option {
	return func(o *options) {
		o.verbose = v
	}
}

type options struct {
	verbose bool
}

// Compile evaluates a Flux script producing a query Spec.
// now parameter must be non-zero, that is the default now time should be set before compiling.
func Compile(ctx context.Context, q string, now time.Time, opts ...Option) (*Spec, error) {
	o := new(options)
	for _, opt := range opts {
		opt(o)
	}

	s, _ := opentracing.StartSpanFromContext(ctx, "parse")
	itrp := NewInterpreter()
	itrp.SetOption(nowOption, nowFunc(now))
	if err := Eval(itrp, q); err != nil {
		return nil, err
	}
	s.Finish()
	s, _ = opentracing.StartSpanFromContext(ctx, "compile")
	defer s.Finish()

	spec := toSpecFromSideEffecs(itrp)

	if o.verbose {
		log.Println("Query Spec: ", Formatted(spec, FmtJSON))
	}
	return spec, nil
}

// Eval evaluates the flux string q and update the given interpreter
func Eval(itrp *interpreter.Interpreter, q string) error {
	astProg, err := parser.NewAST(q)
	if err != nil {
		return err
	}

	_, decls := builtIns(itrp)

	// Convert AST program to a semantic program
	semProg, err := semantic.New(astProg, decls)
	if err != nil {
		return err
	}

	if err := itrp.Eval(semProg); err != nil {
		return err
	}
	return nil
}

// NewInterpreter returns an interpreter instance with
// pre-constructed options and global scopes.
func NewInterpreter() *interpreter.Interpreter {
	options := make(map[string]values.Value, len(builtinOptions))
	globals := make(map[string]values.Value, len(builtinScope))

	for k, v := range builtinScope {
		globals[k] = v
	}

	for k, v := range builtinOptions {
		options[k] = v
	}

	return interpreter.NewInterpreter(options, globals)
}

func nowFunc(now time.Time) values.Function {
	timeVal := values.NewTimeValue(values.ConvertTime(now))
	ftype := semantic.NewFunctionType(semantic.FunctionSignature{
		ReturnType: semantic.Time,
	})
	call := func(args values.Object) (values.Value, error) {
		return timeVal, nil
	}
	sideEffect := false
	return values.NewFunction(nowOption, ftype, call, sideEffect)
}

func toSpecFromSideEffecs(itrp *interpreter.Interpreter) *Spec {
	return ToSpec(itrp, itrp.SideEffects()...)
}

// ToSpec creates a query spec from the interpreter and list of values.
func ToSpec(itrp *interpreter.Interpreter, vals ...values.Value) *Spec {
	ider := &ider{
		id:     0,
		lookup: make(map[*TableObject]OperationID),
	}

	spec := new(Spec)
	visited := make(map[*TableObject]bool)
	nodes := make([]*TableObject, 0, len(vals))

	for _, val := range vals {
		if op, ok := val.(*TableObject); ok {
			dup := false
			for _, node := range nodes {
				if op.Equal(node) {
					dup = true
					break
				}
			}
			if !dup {
				op.buildSpec(ider, spec, visited)
				nodes = append(nodes, op)
			}
		}
	}

	// now option is Time value
	nowValue, _ := itrp.Option(nowOption).Function().Call(nil)
	spec.Now = nowValue.Time().Time()

	return spec
}

type CreateOperationSpec func(args Arguments, a *Administration) (OperationSpec, error)

var builtinScope = make(map[string]values.Value)

// TODO(Josh): Default option values should be registered similarly to built-in
// functions. Default options should be registered in their own files
// (or in a single file) using the RegisterBuiltInOption function which will
// place the resolved option value in the following map.
var builtinOptions = make(map[string]values.Value)
var builtinDeclarations = make(semantic.DeclarationScope)

// list of builtin scripts
var builtins = make(map[string]string)
var finalized bool

// RegisterBuiltIn adds any variable declarations in the script to the builtin scope.
func RegisterBuiltIn(name, script string) {
	if finalized {
		panic(errors.New("already finalized, cannot register builtin"))
	}
	builtins[name] = script
}

// RegisterFunction adds a new builtin top level function.
func RegisterFunction(name string, c CreateOperationSpec, sig semantic.FunctionSignature) {
	f := function{
		t:             semantic.NewFunctionType(sig),
		name:          name,
		createOpSpec:  c,
		hasSideEffect: false,
	}
	RegisterBuiltInValue(name, &f)
}

// RegisterFunctionWithSideEffect adds a new builtin top level function that produces side effects.
// For example, the builtin functions yield(), toKafka(), and toHTTP() all produce side effects.
func RegisterFunctionWithSideEffect(name string, c CreateOperationSpec, sig semantic.FunctionSignature) {
	f := function{
		t:             semantic.NewFunctionType(sig),
		name:          name,
		createOpSpec:  c,
		hasSideEffect: true,
	}
	RegisterBuiltInValue(name, &f)
}

// RegisterBuiltInValue adds the value to the builtin scope.
func RegisterBuiltInValue(name string, v values.Value) {
	if finalized {
		panic(errors.New("already finalized, cannot register builtin"))
	}
	if _, ok := builtinScope[name]; ok {
		panic(fmt.Errorf("duplicate registration for builtin %q", name))
	}
	builtinDeclarations[name] = semantic.NewExternalVariableDeclaration(name, v.Type())
	builtinScope[name] = v
}

// RegisterBuiltInOption adds the value to the builtin scope.
func RegisterBuiltInOption(name string, v values.Value) {
	if finalized {
		panic(errors.New("already finalized, cannot register builtin option"))
	}
	if _, ok := builtinOptions[name]; ok {
		panic(fmt.Errorf("duplicate registration for builtin option %q", name))
	}
	builtinOptions[name] = v
}

// FinalizeBuiltIns must be called to complete registration.
// Future calls to RegisterFunction, RegisterBuiltIn or RegisterBuiltInValue will panic.
func FinalizeBuiltIns() {
	if finalized {
		panic("already finalized")
	}
	finalized = true
	// Call BuiltIns to validate all built-in values are valid.
	// A panic will occur if any value is invalid.
	_, _ = BuiltIns()
}

var TableObjectType = semantic.NewObjectType(map[string]semantic.Type{
	tableKindKey: semantic.String,
	// TODO(nathanielc): The spec types vary significantly making type comparisons impossible, for now the solution is to state the type as an empty object.
	//tableSpecKey: semantic.EmptyObject,
	// TODO(nathanielc): Support recursive types, for now we state that the array has empty objects.
	tableParentsKey: semantic.NewArrayType(semantic.EmptyObject),
})

// IDer produces the mapping of table Objects to OpertionIDs
type IDer interface {
	ID(*TableObject) OperationID
}

// IDerOpSpec is the interface any operation spec that needs
// access to OperationIDs in the query spec must implement.
type IDerOpSpec interface {
	IDer(ider IDer)
}

type TableObject struct {
	// TODO(Josh): Remove args once the
	// OperationSpec interface has an Equal method.
	args    Arguments
	Kind    OperationKind
	Spec    OperationSpec
	Parents values.Array
}

func (t *TableObject) Operation(ider IDer) *Operation {
	if iderOpSpec, ok := t.Spec.(IDerOpSpec); ok {
		iderOpSpec.IDer(ider)
	}

	return &Operation{
		ID:   ider.ID(t),
		Spec: t.Spec,
	}
}

type ider struct {
	id     int
	lookup map[*TableObject]OperationID
}

func (i *ider) nextID() int {
	next := i.id
	i.id++
	return next
}

func (i *ider) get(t *TableObject) (OperationID, bool) {
	tableID, ok := i.lookup[t]
	return tableID, ok
}

func (i *ider) set(t *TableObject, id int) OperationID {
	opID := OperationID(fmt.Sprintf("%s%d", t.Kind, id))
	i.lookup[t] = opID
	return opID
}

func (i *ider) ID(t *TableObject) OperationID {
	tableID, ok := i.get(t)
	if !ok {
		tableID = i.set(t, i.nextID())
	}
	return tableID
}

func (t *TableObject) ToSpec() *Spec {
	ider := &ider{
		id:     0,
		lookup: make(map[*TableObject]OperationID),
	}
	spec := new(Spec)
	visited := make(map[*TableObject]bool)
	t.buildSpec(ider, spec, visited)
	return spec
}

func (t *TableObject) buildSpec(ider IDer, spec *Spec, visited map[*TableObject]bool) {
	// Traverse graph upwards to first unvisited node.
	// Note: parents are sorted based on parameter name, so the visit order is consistent.
	t.Parents.Range(func(i int, v values.Value) {
		p := v.(*TableObject)
		if !visited[p] {
			// rescurse up parents
			p.buildSpec(ider, spec, visited)
		}
	})

	// Assign ID to table object after visiting all ancestors.
	tableID := ider.ID(t)

	// Link table object to all parents after assigning ID.
	t.Parents.Range(func(i int, v values.Value) {
		p := v.(*TableObject)
		spec.Edges = append(spec.Edges, Edge{
			Parent: ider.ID(p),
			Child:  tableID,
		})
	})

	visited[t] = true
	spec.Operations = append(spec.Operations, t.Operation(ider))
}

func (t *TableObject) Type() semantic.Type {
	return TableObjectType
}

func (t *TableObject) Str() string {
	panic(values.UnexpectedKind(semantic.Object, semantic.String))
}
func (t *TableObject) Int() int64 {
	panic(values.UnexpectedKind(semantic.Object, semantic.Int))
}
func (t *TableObject) UInt() uint64 {
	panic(values.UnexpectedKind(semantic.Object, semantic.UInt))
}
func (t *TableObject) Float() float64 {
	panic(values.UnexpectedKind(semantic.Object, semantic.Float))
}
func (t *TableObject) Bool() bool {
	panic(values.UnexpectedKind(semantic.Object, semantic.Bool))
}
func (t *TableObject) Time() values.Time {
	panic(values.UnexpectedKind(semantic.Object, semantic.Time))
}
func (t *TableObject) Duration() values.Duration {
	panic(values.UnexpectedKind(semantic.Object, semantic.Duration))
}
func (t *TableObject) Regexp() *regexp.Regexp {
	panic(values.UnexpectedKind(semantic.Object, semantic.Regexp))
}
func (t *TableObject) Array() values.Array {
	panic(values.UnexpectedKind(semantic.Object, semantic.Array))
}
func (t *TableObject) Object() values.Object {
	return t
}
func (t *TableObject) Equal(rhs values.Value) bool {
	if t.Type() != rhs.Type() {
		return false
	}
	r := rhs.Object()
	if t.Len() != r.Len() {
		return false
	}
	for _, k := range t.keys() {
		val1, ok1 := t.Get(k)
		val2, ok2 := r.Get(k)
		if !ok1 || !ok2 || !val1.Equal(val2) {
			return false
		}
	}
	return true
}
func (t *TableObject) Function() values.Function {
	panic(values.UnexpectedKind(semantic.Object, semantic.Function))
}

func (t *TableObject) Get(name string) (values.Value, bool) {
	switch name {
	case tableKindKey:
		return values.NewStringValue(string(t.Kind)), true
	case tableParentsKey:
		return t.Parents, true
	default:
		return t.args.Get(name)
	}
}

func (t *TableObject) keys() []string {
	tableKeys := make([]string, 0, len(t.args.GetAll())+2)
	return append(tableKeys, tableParentsKey, tableParentsKey)
}

func (t *TableObject) Set(name string, v values.Value) {
	// immutable
}

func (t *TableObject) Len() int {
	return len(t.keys())
}

func (t *TableObject) Range(f func(name string, v values.Value)) {
	for _, arg := range t.args.GetAll() {
		val, _ := t.args.Get(arg)
		f(arg, val)
	}
	f(tableKindKey, values.NewStringValue(string(t.Kind)))
	f(tableParentsKey, t.Parents)
}

// DefaultFunctionSignature returns a FunctionSignature for standard functions which accept a table piped argument.
// It is safe to modify the returned signature.
func DefaultFunctionSignature() semantic.FunctionSignature {
	return semantic.FunctionSignature{
		Params: map[string]semantic.Type{
			TableParameter: TableObjectType,
		},
		ReturnType:   TableObjectType,
		PipeArgument: TableParameter,
	}
}

func BuiltIns() (map[string]values.Value, semantic.DeclarationScope) {
	if !finalized {
		panic("builtins not finalized")
	}
	return builtIns(NewInterpreter())
}

func builtIns(itrp *interpreter.Interpreter) (map[string]values.Value, semantic.DeclarationScope) {
	decls := builtinDeclarations.Copy()

	for name, script := range builtins {
		astProg, err := parser.NewAST(script)
		if err != nil {
			panic(errors.Wrapf(err, "failed to parse builtin %q", name))
		}
		semProg, err := semantic.New(astProg, decls)
		if err != nil {
			panic(errors.Wrapf(err, "failed to create semantic graph for builtin %q", name))
		}

		if err := itrp.Eval(semProg); err != nil {
			panic(errors.Wrapf(err, "failed to evaluate builtin %q", name))
		}
	}
	return itrp.GlobalScope().Values(), decls
}

type Administration struct {
	parents values.Array
}

func newAdministration() *Administration {
	return &Administration{
		// TODO(nathanielc): Once we can support recursive types change this to,
		// interpreter.NewArray(TableObjectType)
		parents: values.NewArray(semantic.EmptyObject),
	}
}

// AddParentFromArgs reads the args for the `table` argument and adds the value as a parent.
func (a *Administration) AddParentFromArgs(args Arguments) error {
	parent, err := args.GetRequiredObject(TableParameter)
	if err != nil {
		return err
	}
	p, ok := parent.(*TableObject)
	if !ok {
		return fmt.Errorf("argument is not a table object: got %T", parent)
	}
	a.AddParent(p)
	return nil
}

// AddParent instructs the evaluation Context that a new edge should be created from the parent to the current operation.
// Duplicate parents will be removed, so the caller need not concern itself with which parents have already been added.
func (a *Administration) AddParent(np *TableObject) {
	// Check for duplicates
	found := false
	a.parents.Range(func(i int, v values.Value) {
		if p, ok := v.(*TableObject); ok && p == np {
			found = true
		}
	})
	if !found {
		a.parents.Append(np)
	}
}

type function struct {
	name          string
	t             semantic.Type
	createOpSpec  CreateOperationSpec
	hasSideEffect bool
}

func (f *function) Type() semantic.Type {
	return f.t
}
func (f *function) Str() string {
	panic(values.UnexpectedKind(semantic.Function, semantic.String))
}
func (f *function) Int() int64 {
	panic(values.UnexpectedKind(semantic.Function, semantic.Int))
}
func (f *function) UInt() uint64 {
	panic(values.UnexpectedKind(semantic.Function, semantic.UInt))
}
func (f *function) Float() float64 {
	panic(values.UnexpectedKind(semantic.Function, semantic.Float))
}
func (f *function) Bool() bool {
	panic(values.UnexpectedKind(semantic.Function, semantic.Bool))
}
func (f *function) Time() values.Time {
	panic(values.UnexpectedKind(semantic.Function, semantic.Time))
}
func (f *function) Duration() values.Duration {
	panic(values.UnexpectedKind(semantic.Function, semantic.Duration))
}
func (f *function) Regexp() *regexp.Regexp {
	panic(values.UnexpectedKind(semantic.Function, semantic.Regexp))
}
func (f *function) Array() values.Array {
	panic(values.UnexpectedKind(semantic.Function, semantic.Array))
}
func (f *function) Object() values.Object {
	panic(values.UnexpectedKind(semantic.Function, semantic.Object))
}
func (f *function) Function() values.Function {
	return f
}
func (f *function) Equal(rhs values.Value) bool {
	if f.Type() != rhs.Type() {
		return false
	}
	v, ok := rhs.(*function)
	return ok && (f == v)
}
func (f *function) HasSideEffect() bool {
	return f.hasSideEffect
}

func (f *function) Call(argsObj values.Object) (values.Value, error) {
	return interpreter.DoFunctionCall(f.call, argsObj)
}

func (f *function) call(args interpreter.Arguments) (values.Value, error) {
	a := newAdministration()
	arguments := Arguments{Arguments: args}
	spec, err := f.createOpSpec(arguments, a)
	if err != nil {
		return nil, err
	}

	t := &TableObject{
		args:    arguments,
		Kind:    spec.Kind(),
		Spec:    spec,
		Parents: a.parents,
	}
	return t, nil
}

type specValue struct {
	spec OperationSpec
}

func (v specValue) Type() semantic.Type {
	return semantic.EmptyObject
}

func (v specValue) Value() interface{} {
	return v.spec
}

func (v specValue) Property(name string) (interpreter.Value, error) {
	return nil, errors.New("spec does not have properties")
}

type Arguments struct {
	interpreter.Arguments
}

func (a Arguments) GetTime(name string) (Time, bool, error) {
	v, ok := a.Get(name)
	if !ok {
		return Time{}, false, nil
	}
	qt, err := ToQueryTime(v)
	if err != nil {
		return Time{}, ok, err
	}
	return qt, ok, nil
}

func (a Arguments) GetRequiredTime(name string) (Time, error) {
	qt, ok, err := a.GetTime(name)
	if err != nil {
		return Time{}, err
	}
	if !ok {
		return Time{}, fmt.Errorf("missing required keyword argument %q", name)
	}
	return qt, nil
}

func (a Arguments) GetDuration(name string) (Duration, bool, error) {
	v, ok := a.Get(name)
	if !ok {
		return 0, false, nil
	}
	return Duration(v.Duration()), true, nil
}

func (a Arguments) GetRequiredDuration(name string) (Duration, error) {
	d, ok, err := a.GetDuration(name)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, fmt.Errorf("missing required keyword argument %q", name)
	}
	return d, nil
}

func ToQueryTime(value values.Value) (Time, error) {
	switch value.Type().Kind() {
	case semantic.Time:
		return Time{
			Absolute: value.Time().Time(),
		}, nil
	case semantic.Duration:
		return Time{
			Relative:   value.Duration().Duration(),
			IsRelative: true,
		}, nil
	case semantic.Int:
		return Time{
			Absolute: time.Unix(value.Int(), 0),
		}, nil
	default:
		return Time{}, fmt.Errorf("value is not a time, got %v", value.Type())
	}
}
