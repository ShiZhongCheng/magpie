package eval

import (
	"bufio"
	"fmt"
	"magpie/ast"
	"magpie/lexer"
	"magpie/parser"
	"magpie/message"
	"os"
	"strconv"
	"strings"
)

const (
	LineStep = 5
)

type Debugger struct {
	SrcLines []string

	Functions map[string]*ast.FunctionLiteral

	//for line number breakpoint
	Breakpoints map[int]bool

	Node ast.Node
	Scope *Scope

	Stepping bool

	prevCommand string
	showPrompt bool
	listLine int
}

func NewDebugger(lines []string) *Debugger {
	d := &Debugger{SrcLines: lines}
	d.Breakpoints = make(map[int]bool)
	d.showPrompt = true
	d.Stepping = true
	d.prevCommand = ""

	return d
}

// Add a breakpoint at source line
func (d *Debugger) AddBP(line int) {
	d.Breakpoints[line] = true
}

// Delete a breakpoint at source line
func (d *Debugger) DelBP(line int) {
	if _, ok := d.Breakpoints[line]; ok {
		delete(d.Breakpoints, line)
	}
}

// Check if a source line is at a breakpoint
func (d *Debugger) IsBP(line int) bool {
	_, ok := d.Breakpoints[line];
	return ok
}

func (d * Debugger) SetNodeAndScope(node ast.Node, scope *Scope) {
	d.Node = node
	d.Scope = scope
}

func (d * Debugger) SetFunctions(functions map[string]*ast.FunctionLiteral) {
	d.Functions = functions
}

func (d * Debugger) ShowBanner() {
	fmt.Println("                                    _     ")
	fmt.Println("   ____ ___   ____ _ ____ _ ____   (_)___ ")
	fmt.Println("  / __ `__ \\ / __ `// __ `// __ \\ / // _ \\")
	fmt.Println(" / / / / / // /_/ // /_/ // /_/ // //  __/")
	fmt.Println("/_/ /_/ /_/ \\__,_/ \\__, // .___//_/ \\___/ ")
	fmt.Println("                  /____//_/             ");
	fmt.Println("");
}

func (d *Debugger) ProcessCommand() {
	for {
		if !d.showPrompt {
			break
		}

		p := d.Node.Pos()

		fmt.Printf("%d\t\t%s", p.Line, d.SrcLines[p.Line])
		fmt.Print("\n(magpie) ")

		fmt.Print("\x1b[1m\x1b[36m")

		reader := bufio.NewReader(os.Stdin)
		command, _ := reader.ReadString('\n')
		command = strings.Trim(command, "\r\n")
		if command == "" && d.prevCommand != "" {
			command = d.prevCommand
		}

		fmt.Print("\x1b[0m")

		d.prevCommand = command

		d.Stepping = false
		if strings.Compare("c", command) == 0 || strings.Compare("continue", command) == 0 {
			break
		} else if strings.Compare("n", command) == 0 || strings.Compare("next", command) == 0 {
			d.Stepping = true
			break
		} else if strings.HasPrefix(command, "b ")|| strings.HasPrefix(command, "bp ") {
			arr := strings.Split(command, " ")
			if len(arr) < 2 {
				fmt.Println("Line number or function name expected.")
			} else {
				line, err := strconv.Atoi(arr[1])
				if err == nil {
					if line <= 0 {
						fmt.Println("Line number must greater than zero.")
					} else {
						d.AddBP(line)
					}
				} else {
					funcName := arr[1]
					var f *ast.FunctionLiteral
					var ok bool
					if f, ok = d.Functions[funcName]; !ok {
						fmt.Println("Function name not found.")
					} else {
						d.AddBP(f.StmtPos().Line)
					}
				}
			}

		} else if strings.HasPrefix(command, "d ") || strings.HasPrefix(command, "del ") {
			arr := strings.Split(command, " ")
			if len(arr) < 2 {
				fmt.Println("Line number expected.")
			} else {
				line, err := strconv.Atoi(arr[1])
				if err == nil {
					if line <= 0 {
						fmt.Println("Line number must greater than zero.")
					} else {
						d.DelBP(line)
					}
				} else {
					funcName := arr[1]
					var f *ast.FunctionLiteral
					var ok bool
					if f, ok = d.Functions[funcName]; !ok {
						fmt.Println("Function name not found.")
					} else {
						d.DelBP(f.StmtPos().Line)
					}
				}
			}

		} else if strings.HasPrefix(command, "p ") || strings.HasPrefix(command, "print ") ||
			strings.HasPrefix(command, "e ") || strings.HasPrefix(command, "eval ") {
			exp := strings.Split(command, " ")[1:]
			lex := lexer.New("", strings.Join(exp, ""))
			wd, _ := os.Getwd()
			p := parser.New(lex, wd)
			oldLines := d.SrcLines
			oldNode := d.Node
			d.showPrompt = false
			program := p.ParseProgram()
			aval := Eval(program, d.Scope)
			fmt.Printf("%s\n\n", aval.Inspect())
			d.SrcLines = oldLines
			d.Node = oldNode
			d.showPrompt = true
		} else if strings.Compare("exit", command) == 0 || strings.Compare("quit", command) == 0 ||
				  strings.Compare("bye", command) == 0 || strings.Compare("q", command) == 0 {
			os.Exit(0)
		} else if strings.Compare("l", command) == 0 || strings.Compare("list", command) == 0 {
			if d.listLine == 0 {
				d.listLine = p.Line
			}

			if d.listLine < len(d.SrcLines) {
				for i := d.listLine; i <= d.listLine + LineStep; i++ {
					if i >= len(d.SrcLines) {
						break
					}
					fmt.Printf("\n%d\t\t%s", i, d.SrcLines[i])
				}
				fmt.Println()
			}

			d.listLine = d.listLine + LineStep + 1
			if d.listLine >= len(d.SrcLines) {
				d.listLine = 0
			}
		} else {
			fmt.Printf("Undefined command: '%s'.  Try 'help'.\n", command)
		}
	} //end for
}

//Check if node can be stopped, some nodes cannot be stopped, 
//e.g. 'InfixExpression', 'IntegerLiteral'
func (d *Debugger) CanStop() bool {
	flag := false
	switch n := d.Node.(type) {
	case *ast.LetStatement:
		if !n.InClass {
			flag = true
		}
	case *ast.ConstStatement:
		flag = true
	case *ast.ReturnStatement:
		flag = true
	case *ast.DeferStmt:
		flag = true
	case *ast.EnumStatement:
		flag = true
	case *ast.IfExpression:
		flag = true
	case *ast.UnlessExpression:
		flag = true
	case *ast.CaseExpr:
		flag = true
	case *ast.DoLoop:
		flag = true
	case *ast.WhileLoop:
		flag = true
	case *ast.ForLoop:
		flag = true
	case *ast.ForEverLoop:
		flag = true
	case *ast.ForEachArrayLoop:
		flag = true
	case *ast.ForEachDotRange:
		flag = true
	case *ast.ForEachMapLoop:
		flag = true
	case *ast.BreakExpression:
		flag = true
	case *ast.ContinueExpression:
		flag = true
	case *ast.AssignExpression:
		flag = true
	case *ast.CallExpression:
		flag = true
	case *ast.TryStmt:
		flag = true
	case *ast.SpawnStmt:
		flag = true
	case *ast.UsingStmt:
		flag = true
	case *ast.QueryExpr:
		flag = true
	default:
		flag = false
	}

	return flag
}

func (d *Debugger) MessageReceived(msg message.Message) {
	ctx := msg.Body.(Context)

	msgType := msg.Type
	switch (msgType) {
	case message.EVAL_LINE:
		line := ctx.N[0].Pos().Line;
			if d.Stepping {
			d.ProcessCommand()
		} else if (d.IsBP(line)) {
			fmt.Printf("\nBreakpoint hit at line %d\n", line)
			d.ProcessCommand()
		}

	case message.CALL:
		// c := ctx.N[0].(*ast.CallExpression)
		// fn := c.Function.String()
		// for funcName, f := range d.Functions {
		// 	if fn == funcName {
		// 		fmt.Printf("\nEnter function '%s' at line %d\n", fn, f.StmtPos().Line)
		// 		break
		// 	}
		// }
	case message.METHOD_CALL:
		// mc := ctx.N[0].(*ast.MethodCallExpression)
		// obj := mc.Object.String()
		// if call, ok := mc.Call.(*ast.CallExpression); ok {
		// 	fn := call.Function.String()
		// 	for funcName, f := range d.Functions {
		// 		if fn == funcName {
		// 			fmt.Printf("\nEnter function '%s.%s' at line %d\n", obj, fn, f.StmtPos().Line)
		// 			break
		// 		}
		// 	}
		// }

	case message.RETURN:
		// r := ctx.N[0].(*ast.ReturnStatement)
		// line := r.Pos().Line
		// for funcName, f := range d.Functions {
		// 	if line >= f.Pos().Line && line <= f.End().Line {
		// 		fmt.Printf("Function '%s' returns\n\n", funcName)
		// 		break
		// 	}
		// }
	}
}