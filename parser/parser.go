package parser

import (
	"fmt"
	"interpreter/ast"
	"interpreter/lexer"
	"interpreter/token"
	"strconv"
)

const (
	// 优先级
	_ int = iota
	LOWEST
	EQUALS      // ==
	LESSGREATER // < or >
	SUM         // +
	PRODUCT     // *
	PREFIX      // -X or !X
	CALL        // add(X)
	INDEX       // arr[i]
)

var (
	// 优先级表
	precedences = map[token.Type]int{
		token.EQ:       EQUALS,
		token.NEQ:      EQUALS,
		token.LT:       LESSGREATER,
		token.GT:       LESSGREATER,
		token.PLUS:     SUM,
		token.MINUS:    SUM,
		token.SLASH:    PRODUCT,
		token.ASTERISK: PRODUCT,
		token.LPAREN:   CALL,
		token.LBRACKET: INDEX,
	}
)

type (
	// 本例不实现后缀表达式 a++
	prefixParseFn func() ast.Expression                        // 前缀表达式解析 !true -2
	infixParseFn  func(leftExpr ast.Expression) ast.Expression // 中缀表达式解析 1+2 a!=b
)

type Parser struct {
	l              *lexer.Lexer // 词法分析器
	curToken       token.Token  // 当前
	peekToken      token.Token  // 下一个，当cur没有足够信息来判断是，需要借助peek
	errors         []string     // 解析过程中遇到的错误
	prefixParseFns map[token.Type]prefixParseFn
	infixParseFns  map[token.Type]infixParseFn
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		l:              l,
		errors:         []string{},
		prefixParseFns: map[token.Type]prefixParseFn{},
		infixParseFns:  map[token.Type]infixParseFn{},
	}
	p.registerPrefix(token.IDENT, p.parseIdentifier)
	p.registerPrefix(token.INT, p.parseIntegerLiteral)
	p.registerPrefix(token.STRING, p.parseStringLiteral)
	p.registerPrefix(token.LBRACKET, p.parseArrayLiteral)
	p.registerPrefix(token.TRUE, p.parseBoolean)
	p.registerPrefix(token.FALSE, p.parseBoolean)
	p.registerPrefix(token.BANG, p.parsePrefixExpression)
	p.registerPrefix(token.MINUS, p.parsePrefixExpression)
	p.registerPrefix(token.LPAREN, p.parseGroupedExpression)
	p.registerPrefix(token.IF, p.parseIfExpression)
	p.registerPrefix(token.FUNCTION, p.parseFunctionLiteral)
	p.registerPrefix(token.LBRACE, p.parseHashLiteral)
	p.registerInfix(token.PLUS, p.parseInfixExpression)
	p.registerInfix(token.MINUS, p.parseInfixExpression)
	p.registerInfix(token.ASTERISK, p.parseInfixExpression)
	p.registerInfix(token.SLASH, p.parseInfixExpression)
	p.registerInfix(token.EQ, p.parseInfixExpression)
	p.registerInfix(token.NEQ, p.parseInfixExpression)
	p.registerInfix(token.LT, p.parseInfixExpression)
	p.registerInfix(token.GT, p.parseInfixExpression)
	// 函数调用 <FunctionLiteral>(...) 所以为(注册中缀解析
	p.registerInfix(token.LPAREN, p.parseCallExpression)
	// 索引调用 <ArrayLiteral>[...]
	p.registerInfix(token.LBRACKET, p.parseIndexExpression)
	// 读2次，给cur和peek赋初始值
	// 1. cur变为nil peek变为头
	// 2. cur变为头 peek变为下一个
	p.nextToken()
	p.nextToken()
	return p
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{}
	program.Statements = []ast.Statement{}
	// 反复读取直到结尾
	for p.curToken.Type != token.EOF {
		// 解析表达式
		stmt := p.parseStatement()
		// 如果能解析出表达式
		if stmt != nil {
			// 那把它放进program的集合里
			program.Statements = append(program.Statements, stmt)
		}
		// 下一个
		p.nextToken()
	}
	return program
}

func (p *Parser) parseStatement() ast.Statement {
	switch p.curToken.Type {
	case token.LET:
		return p.parseLetStatement()
	case token.RETURN:
		return p.parseReturnStatement()
	default:
		return p.parseExpressionStatement()
	}
}

func (p *Parser) parseLetStatement() *ast.LetStatement {
	stmt := &ast.LetStatement{Token: p.curToken}
	// 当前是let，所以下一个应当是 IDENT 标识符
	if !p.expectPeek(token.IDENT) {
		return nil
	}
	stmt.Name = &ast.Identifier{
		Token: p.curToken,
		Value: p.curToken.Literal,
	}
	// 标识符后应当是赋值
	if !p.expectPeek(token.ASSIGN) {
		return nil
	}
	// 当前是赋值号，跳过到表达式
	p.nextToken()
	stmt.Value = p.parseExpression(LOWEST)
	// 分号或没有
	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}
	return stmt
}

func (p *Parser) parseReturnStatement() *ast.ReturnStatement {
	stmt := &ast.ReturnStatement{Token: p.curToken}
	// 当前是return，推进下一个
	p.nextToken()
	// 表达式
	stmt.ReturnValue = p.parseExpression(LOWEST)
	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}
	return stmt
}

func (p *Parser) parseExpressionStatement() *ast.ExpressionStatement {
	stmt := &ast.ExpressionStatement{Token: p.curToken}
	stmt.Expression = p.parseExpression(LOWEST)
	// 有没有分号都无所谓，如果有分号，那就再读一位
	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}
	return stmt
}

func (p *Parser) parseExpression(precedence int) ast.Expression {
	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.noPrefixParseFnError(p.curToken.Type)
		return nil
	}
	leftExpr := prefix()
	for !p.peekTokenIs(token.SEMICOLON) && precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return leftExpr
		}
		p.nextToken()
		leftExpr = infix(leftExpr)
	}
	return leftExpr
}

func (p *Parser) parseIdentifier() ast.Expression {
	return &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseIntegerLiteral() ast.Expression {
	lit := &ast.IntegerLiteral{Token: p.curToken}
	value, err := strconv.ParseInt(p.curToken.Literal, 0, 64)
	if err != nil {
		msg := fmt.Sprintf("无法解析 %q 为数字", p.curToken.Literal)
		p.errors = append(p.errors, msg)
		return nil
	}
	lit.Value = value
	return lit
}

func (p *Parser) parseStringLiteral() ast.Expression {
	return &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseArrayLiteral() ast.Expression {
	array := &ast.ArrayLiteral{Token: p.curToken}
	array.Elements = p.parseExpressionList(token.RBRACKET)
	return array
}

func (p *Parser) parseIndexExpression(leftExpr ast.Expression) ast.Expression {
	expr := &ast.IndexExpression{Token: p.curToken, Left: leftExpr}
	p.nextToken()
	expr.Index = p.parseExpression(LOWEST)
	if !p.expectPeek(token.RBRACKET) {
		return nil
	}
	return expr
}

func (p *Parser) parseExpressionList(end token.Type) []ast.Expression {
	args := make([]ast.Expression, 0)
	if p.peekTokenIs(end) {
		// 无参调用
		p.nextToken()
		return args
	}
	p.nextToken()
	args = append(args, p.parseExpression(LOWEST))
	for p.peekTokenIs(token.COMMA) {
		// 跳过前一个表达式参数
		p.nextToken()
		// 跳过逗号
		p.nextToken()
		args = append(args, p.parseExpression(LOWEST))
	}
	if !p.expectPeek(end) {
		return nil
	}
	return args
}

func (p *Parser) parseBoolean() ast.Expression {
	return &ast.Boolean{Token: p.curToken, Value: p.curTokenIs(token.TRUE)}
}

func (p *Parser) parseGroupedExpression() ast.Expression {
	// 跳过当前(
	p.nextToken()
	expr := p.parseExpression(LOWEST)
	if !p.expectPeek(token.RPAREN) {
		// 没有)
		return nil
	}
	return expr
}

func (p *Parser) parseIfExpression() ast.Expression {
	expr := &ast.IfExpression{Token: p.curToken}
	if !p.expectPeek(token.LPAREN) {
		// 当前是IF，下一位不是(
		return nil
	}
	// 当前是(，推进到条件表达式
	p.nextToken()
	// 条件表达式
	expr.Condition = p.parseExpression(LOWEST)
	if !p.expectPeek(token.RPAREN) {
		// )
		return nil
	}
	if !p.expectPeek(token.LBRACE) {
		// {
		return nil
	}
	// 成立情况下的表达式，}已经在循环末推进掉了
	expr.Consequence = p.parseBlockStatement()
	if p.peekTokenIs(token.ELSE) {
		p.nextToken()
		if !p.expectPeek(token.LBRACE) {
			// 有else但是没有{
			return nil
		}
		// 否则条件表达式，}已经在循环末推进掉了
		expr.Alternative = p.parseBlockStatement()
	}
	return expr
}

func (p *Parser) parseFunctionLiteral() ast.Expression {
	lit := &ast.FunctionLiteral{
		Token: p.curToken,
	}
	// 当前是fn
	if !p.expectPeek(token.LPAREN) {
		// 下一个不是(
		return nil
	}
	// 解析参数
	lit.Parameters = p.parseFunctionParameters()
	if !p.expectPeek(token.LBRACE) {
		// { 函数体开始
		return nil
	}
	// 解析函数体
	lit.Body = p.parseBlockStatement()
	return lit
}

func (p *Parser) parseFunctionParameters() []*ast.Identifier {
	identifiers := make([]*ast.Identifier, 0)
	if p.peekTokenIs(token.RPAREN) {
		// 无参，推进)
		p.nextToken()
		return identifiers
	}
	// 当前是(，推进一位
	p.nextToken()
	first := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	identifiers = append(identifiers, first)
	for p.peekTokenIs(token.COMMA) {
		// 跳过前一个参数
		p.nextToken()
		// 跳过逗号
		p.nextToken()
		ident := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		identifiers = append(identifiers, ident)
	}
	if !p.expectPeek(token.RPAREN) {
		// 没有)
		return nil
	}
	return identifiers
}

func (p *Parser) parseBlockStatement() *ast.BlockStatement {
	block := &ast.BlockStatement{Token: p.curToken}
	block.Statements = make([]ast.Statement, 0)
	// 当前是{
	p.nextToken()
	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		// 直到 } 或 结尾
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		// 循环的最后会把}推进掉
		p.nextToken()
	}
	return block
}

func (p *Parser) parseHashLiteral() ast.Expression {
	expr := &ast.HashLiteral{Token: p.curToken}
	expr.Pairs = make(map[ast.Expression]ast.Expression)
	for !p.peekTokenIs(token.RBRACE) { // 不进入，那是空哈希
		p.nextToken()
		key := p.parseExpression(LOWEST)
		if !p.expectPeek(token.COLON) {
			return nil
		}
		p.nextToken()
		val := p.parseExpression(LOWEST)
		expr.Pairs[key] = val
		if !p.peekTokenIs(token.RBRACE) && !p.expectPeek(token.COMMA) {
			// 既不是}结束，也不是,下一个 结束
			return nil
		}
	}
	if !p.expectPeek(token.RBRACE) {
		return nil
	}
	return expr
}

func (p *Parser) parsePrefixExpression() ast.Expression {
	expr := &ast.PrefixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
	}
	// 推进解析前缀符号后表达式
	p.nextToken()
	expr.Right = p.parseExpression(PREFIX)
	return expr
}

func (p *Parser) parseInfixExpression(leftExpr ast.Expression) ast.Expression {
	expr := &ast.InfixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
		Left:     leftExpr,
	}
	precedence := p.curPrecedence()
	p.nextToken()
	expr.Right = p.parseExpression(precedence)
	return expr
}

func (p *Parser) parseCallExpression(leftExpr ast.Expression) ast.Expression {
	expr := &ast.CallExpression{Token: p.curToken, Function: leftExpr}
	expr.Arguments = p.parseExpressionList(token.RPAREN)
	return expr
}

func (p *Parser) curTokenIs(t token.Type) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t token.Type) bool {
	return p.peekToken.Type == t
}

// 判断并推进
func (p *Parser) expectPeek(t token.Type) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	} else {
		p.peekError(t)
		return false
	}
}

func (p *Parser) Errors() []string {
	return p.errors
}

func (p *Parser) peekError(t token.Type) {
	msg := fmt.Sprintf("期望下一个token是 %s，但是实际是 %s", t, p.peekToken.Type)
	p.errors = append(p.errors, msg)
}

func (p *Parser) peekPrecedence() int {
	if precedence, ok := precedences[p.peekToken.Type]; ok {
		return precedence
	}
	return LOWEST
}

func (p *Parser) curPrecedence() int {
	if precedence, ok := precedences[p.curToken.Type]; ok {
		return precedence
	}
	return LOWEST
}

func (p *Parser) noPrefixParseFnError(t token.Type) {
	msg := fmt.Sprintf("没有针对 %s 的前缀表达式解析函数", t)
	p.errors = append(p.errors, msg)
}

func (p *Parser) registerPrefix(tokenType token.Type, fn prefixParseFn) {
	p.prefixParseFns[tokenType] = fn
}

func (p *Parser) registerInfix(tokenType token.Type, fn infixParseFn) {
	p.infixParseFns[tokenType] = fn
}
