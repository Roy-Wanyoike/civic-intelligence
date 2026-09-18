package domain

import (
        "fmt"
        "strconv"
        "strings"
)

// EvaluateConstraint evaluates a constraint Expression against the supplied
// scenario variables. Phase 18 §21 (ValidateConstraints gate). Issue #210.
//
// The evaluator supports a deliberately small expression grammar:
//
//   - Comparison operators: <, <=, >, >=, ==, !=
//   - Logical operators:    && (AND), || (OR), ! (unary NOT)
//   - Variable references by name (e.g. tax_rate); resolved against vars.
//   - Numeric literals (integer or decimal), e.g. 0.30, 6, -2.5.
//   - Boolean literals: true, false.
//   - Parenthesised sub-expressions.
//
// Examples:
//
//      tax_rate <= 0.30 && implementation_delay < 6
//      !(tax_rate > 0.50 || funding_level < 1_000_000)
//      interest_rate == 5.0
//
// EvaluateConstraint returns nil if the expression parses and evaluates to
// true. It returns an error if the expression is syntactically invalid,
// references an unknown variable, applies an operator to the wrong type, or
// evaluates to false. The error message is human-readable and identifies the
// failing expression so callers can surface it during ValidateConstraints.
//
// An empty expression is treated as a no-op (returns nil). This keeps the
// evaluator usable for constraints that exist purely for documentation.
func EvaluateConstraint(expr string, vars []ScenarioVariable) error {
        if strings.TrimSpace(expr) == "" {
                return nil
        }
        p := &constraintParser{
                input: strings.TrimSpace(expr),
                vars:  constraintVarMap(vars),
        }
        result, err := p.parseExpression()
        if err != nil {
                return fmt.Errorf("constraint %q: %w", expr, err)
        }
        if err := p.expectEOF(); err != nil {
                return fmt.Errorf("constraint %q: trailing input: %w", expr, err)
        }
        if !result {
                return fmt.Errorf("constraint %q: evaluates to false", expr)
        }
        return nil
}

// constraintVarMap builds a name -> value lookup from scenario variables.
// Non-numeric variable values (e.g. categorical strings) are still
// resolvable; the evaluator only fails if a comparison operator is applied
// to a non-numeric operand.
func constraintVarMap(vars []ScenarioVariable) map[string]any {
        m := make(map[string]any, len(vars))
        for _, v := range vars {
                m[v.Name] = v.Value
        }
        return m
}

// constraintParser is a tiny recursive-descent parser for the constraint
// grammar. It is intentionally hand-rolled to avoid pulling in a parser
// generator dependency; the grammar is small enough that a hand-rolled
// parser is clearer.
type constraintParser struct {
        input string
        pos   int
        vars  map[string]any
}

// peek returns the next byte without consuming it. Returns 0 at EOF.
func (p *constraintParser) peek() byte {
        if p.pos >= len(p.input) {
                return 0
        }
        return p.input[p.pos]
}

// skipWhitespace consumes any run of spaces, tabs, or newlines.
func (p *constraintParser) skipWhitespace() {
        for p.pos < len(p.input) {
                c := p.input[p.pos]
                if c != ' ' && c != '\t' && c != '\n' && c != '\r' {
                        return
                }
                p.pos++
        }
}

// expectEOF returns an error if there is any unconsumed input.
func (p *constraintParser) expectEOF() error {
        p.skipWhitespace()
        if p.pos < len(p.input) {
                return fmt.Errorf("unexpected trailing input at position %d: %q", p.pos, p.input[p.pos:])
        }
        return nil
}

// match returns true and consumes the literal if it appears at the current
// position. Whitespace is skipped first. The literal MUST be one of the
// operators we recognise; multi-char literals are checked before single-char
// ones at the call site to disambiguate (e.g. "<=" before "<").
func (p *constraintParser) match(lit string) bool {
        p.skipWhitespace()
        if strings.HasPrefix(p.input[p.pos:], lit) {
                p.pos += len(lit)
                return true
        }
        return false
}

// matchComparisonOp returns the next comparison operator (if any) and
// consumes it. Longer operators are matched first so "<=" wins over "<".
func (p *constraintParser) matchComparisonOp() (string, bool) {
        p.skipWhitespace()
        rest := p.input[p.pos:]
        // Order matters: two-char operators must be checked before single-char.
        for _, op := range []string{"<=", ">=", "==", "!=", "<", ">"} {
                if strings.HasPrefix(rest, op) {
                        p.pos += len(op)
                        return op, true
                }
        }
        return "", false
}

// parseExpression is the entry point. Delegates to the OR level.
func (p *constraintParser) parseExpression() (bool, error) {
        return p.parseOr()
}

// parseOr: or_expr := and_expr ('||' and_expr)*
func (p *constraintParser) parseOr() (bool, error) {
        left, err := p.parseAnd()
        if err != nil {
                return false, err
        }
        for p.match("||") {
                right, err := p.parseAnd()
                if err != nil {
                        return false, err
                }
                left = left || right
        }
        return left, nil
}

// parseAnd: and_expr := not_expr ('&&' not_expr)*
func (p *constraintParser) parseAnd() (bool, error) {
        left, err := p.parseNot()
        if err != nil {
                return false, err
        }
        for p.match("&&") {
                right, err := p.parseNot()
                if err != nil {
                        return false, err
                }
                left = left && right
        }
        return left, nil
}

// parseNot handles the unary prefix '!'. A '!' that is immediately followed
// by '=' is the '!=' comparison operator and must be left for parseComparison
// to handle. We only consume '!' here when it is NOT followed by '='.
func (p *constraintParser) parseNot() (bool, error) {
        p.skipWhitespace()
        if p.pos+1 < len(p.input) && p.input[p.pos] == '!' && p.input[p.pos+1] != '=' {
                p.pos++ // consume '!'
                v, err := p.parseNot()
                if err != nil {
                        return false, err
                }
                return !v, nil
        }
        return p.parseComparison()
}

// parseComparison parses a primary, optionally followed by a comparison
// operator and another primary. If no operator follows, the primary MUST
// itself be boolean (e.g. a parenthesised sub-expression or a boolean
// literal). A bare numeric value with no comparison is rejected.
func (p *constraintParser) parseComparison() (bool, error) {
        left, err := p.parsePrimary()
        if err != nil {
                return false, err
        }
        op, ok := p.matchComparisonOp()
        if !ok {
                // No comparison operator — left must be a boolean value.
                b, ok := constraintToBool(left)
                if !ok {
                        return false, fmt.Errorf("expected boolean expression but got %v at position %d", left, p.pos)
                }
                return b, nil
        }
        right, err := p.parsePrimary()
        if err != nil {
                return false, err
        }
        return p.applyComparison(op, left, right)
}

// parsePrimary parses a number, an identifier (variable or boolean literal),
// or a parenthesised sub-expression.
func (p *constraintParser) parsePrimary() (any, error) {
        p.skipWhitespace()
        c := p.peek()
        if c == '(' {
                p.pos++ // consume '('
                v, err := p.parseExpression()
                if err != nil {
                        return nil, err
                }
                p.skipWhitespace()
                if p.peek() != ')' {
                        return nil, fmt.Errorf("expected ')' at position %d", p.pos)
                }
                p.pos++ // consume ')'
                return v, nil
        }
        if isDigit(c) || c == '-' || c == '+' {
                return p.parseNumber()
        }
        if isAlpha(c) || c == '_' {
                return p.parseIdentifier()
        }
        return nil, fmt.Errorf("unexpected character %q at position %d", string(c), p.pos)
}

// parseNumber consumes a numeric literal and returns it as float64. We always
// return float64 (even for integer literals) to keep arithmetic uniform across
// int/decimal variables; comparison operators coerce both sides to float64.
func (p *constraintParser) parseNumber() (any, error) {
        start := p.pos
        if p.peek() == '-' || p.peek() == '+' {
                p.pos++
        }
        hasDigits := false
        for p.pos < len(p.input) && isDigit(p.input[p.pos]) {
                p.pos++
                hasDigits = true
        }
        if p.pos < len(p.input) && p.input[p.pos] == '.' {
                p.pos++
                for p.pos < len(p.input) && isDigit(p.input[p.pos]) {
                        p.pos++
                }
        }
        if !hasDigits {
                return nil, fmt.Errorf("invalid number literal at position %d", start)
        }
        s := p.input[start:p.pos]
        // ParseFloat handles both integer and decimal literals uniformly; we
        // always return float64 so arithmetic across int/decimal variables is
        // consistent without per-type special cases.
        f, err := strconv.ParseFloat(s, 64)
        if err != nil {
                return nil, fmt.Errorf("invalid number literal %q: %w", s, err)
        }
        return f, nil
}

// parseIdentifier consumes an identifier and resolves it. The reserved words
// "true" and "false" are boolean literals; any other identifier is looked up
// in the variable map.
func (p *constraintParser) parseIdentifier() (any, error) {
        start := p.pos
        for p.pos < len(p.input) && (isAlpha(p.input[p.pos]) || isDigit(p.input[p.pos]) || p.input[p.pos] == '_') {
                p.pos++
        }
        name := p.input[start:p.pos]
        switch name {
        case "true":
                return true, nil
        case "false":
                return false, nil
        }
        v, ok := p.vars[name]
        if !ok {
                return nil, fmt.Errorf("unknown variable %q", name)
        }
        return v, nil
}

// applyComparison evaluates a single comparison. Both operands must coerce to
// float64 — applying a comparison operator to a non-numeric operand is an
// error (the constraint is malformed).
func (p *constraintParser) applyComparison(op string, leftAny, rightAny any) (bool, error) {
        left, ok := toFloat(leftAny)
        if !ok {
                return false, fmt.Errorf("operator %s: left operand %v is not numeric", op, leftAny)
        }
        right, ok := toFloat(rightAny)
        if !ok {
                return false, fmt.Errorf("operator %s: right operand %v is not numeric", op, rightAny)
        }
        switch op {
        case "<":
                return left < right, nil
        case "<=":
                return left <= right, nil
        case ">":
                return left > right, nil
        case ">=":
                return left >= right, nil
        case "==":
                return left == right, nil
        case "!=":
                return left != right, nil
        }
        return false, fmt.Errorf("unknown operator %s", op)
}

// constraintToBool coerces a value to a bool. Only the literal true/false and
// boolean variables count — there is no implicit "non-zero = true" rule for
// numeric values.
func constraintToBool(v any) (bool, bool) {
        b, ok := v.(bool)
        return b, ok
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

func isAlpha(c byte) bool {
        return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}
