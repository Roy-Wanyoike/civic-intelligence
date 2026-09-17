package domain

import (
        "strings"
        "testing"
)

// makeVars builds a slice of ScenarioVariables for the test cases below.
// Variable values are intentionally typed to cover int, int64, float32 and
// float64 paths so the coercion logic in toFloat is exercised.
func makeVars() []ScenarioVariable {
        return []ScenarioVariable{
                {Name: "tax_rate", Type: VariablePercentage, Value: float64(0.30)},
                {Name: "implementation_delay", Type: VariableInteger, Value: int(3)},
                {Name: "funding_level", Type: VariableCurrency, Value: int64(2_000_000)},
                {Name: "interest_rate", Type: VariableDecimal, Value: float32(5.5)},
                {Name: "is_active", Type: VariableBoolean, Value: true},
        }
}

// TestEvaluateConstraint_AcceptsTrueExpressions verifies that every supported
// operator and logical connective evaluates to true when it should.
func TestEvaluateConstraint_AcceptsTrueExpressions(t *testing.T) {
        vars := makeVars()
        cases := []string{
                "tax_rate <= 0.30",
                "tax_rate == 0.30",
                "tax_rate >= 0.30",
                "tax_rate != 0.50",
                "tax_rate < 0.31",
                "tax_rate > 0.29",
                "tax_rate <= 0.30 && implementation_delay < 6",
                "tax_rate <= 0.30 && implementation_delay < 6 && funding_level > 1000000",
                "tax_rate > 0.50 || implementation_delay < 6",
                "!(tax_rate > 0.50)",
                "(tax_rate <= 0.30) && (implementation_delay < 6)",
                "interest_rate == 5.5",
                "true",
        }
        for _, expr := range cases {
                t.Run(expr, func(t *testing.T) {
                        if err := EvaluateConstraint(expr, vars); err != nil {
                                t.Fatalf("expected nil error for true expression; got %v", err)
                        }
                })
        }
}

// TestEvaluateConstraint_RejectsFalseExpressions verifies that expressions
// which evaluate to false return a non-nil error mentioning "evaluates to
// false".
func TestEvaluateConstraint_RejectsFalseExpressions(t *testing.T) {
        vars := makeVars()
        cases := []string{
                "tax_rate > 0.50",
                "tax_rate == 0.50",
                "tax_rate < 0.10",
                "implementation_delay > 6",
                "tax_rate <= 0.30 && implementation_delay > 6",
                "tax_rate > 0.50 || implementation_delay > 6",
                "!(tax_rate <= 0.30)",
                "false",
        }
        for _, expr := range cases {
                t.Run(expr, func(t *testing.T) {
                        err := EvaluateConstraint(expr, vars)
                        if err == nil {
                                t.Fatal("expected error for false expression; got nil")
                        }
                        if !strings.Contains(err.Error(), "evaluates to false") {
                                t.Errorf("expected error to mention 'evaluates to false'; got %v", err)
                        }
                })
        }
}

// TestEvaluateConstraint_RejectsUnknownVariable verifies that referencing a
// variable that does not exist in the scenario returns a clear parse error.
func TestEvaluateConstraint_RejectsUnknownVariable(t *testing.T) {
        vars := makeVars()
        err := EvaluateConstraint("nonexistent_var <= 0.50", vars)
        if err == nil {
                t.Fatal("expected error for unknown variable; got nil")
        }
        if !strings.Contains(err.Error(), "unknown variable") {
                t.Errorf("expected 'unknown variable' in error; got %v", err)
        }
}

// TestEvaluateConstraint_RejectsMalformedExpressions verifies that parser
// errors are surfaced as errors.
func TestEvaluateConstraint_RejectsMalformedExpressions(t *testing.T) {
        vars := makeVars()
        cases := []string{
                "tax_rate",            // bare variable with no comparison
                "tax_rate <",          // operator without right-hand operand
                "tax_rate <= 0.30 &&", // dangling operator
                "tax_rate <= 0.30 extra", // trailing tokens
                "<= 0.30",             // missing left operand
                "(tax_rate <= 0.30",   // unclosed parenthesis
        }
        for _, expr := range cases {
                t.Run(expr, func(t *testing.T) {
                        err := EvaluateConstraint(expr, vars)
                        if err == nil {
                                t.Fatal("expected error for malformed expression; got nil")
                        }
                })
        }
}

// TestEvaluateConstraint_EmptyExpressionIsNoop verifies that an empty
// constraint expression is treated as a no-op. This lets callers declare
// documentation-only constraints without forcing them to write an
// expression.
func TestEvaluateConstraint_EmptyExpressionIsNoop(t *testing.T) {
        if err := EvaluateConstraint("", makeVars()); err != nil {
                t.Fatalf("expected nil error for empty expression; got %v", err)
        }
        if err := EvaluateConstraint("   \t  ", makeVars()); err != nil {
                t.Fatalf("expected nil error for whitespace-only expression; got %v", err)
        }
}

// TestEvaluateConstraint_IntegerAndFloatCoercion verifies that int, int64,
// float32 and float64 variable values are all coerced uniformly so that
// comparisons across types work correctly.
func TestEvaluateConstraint_IntegerAndFloatCoercion(t *testing.T) {
        vars := makeVars()
        // implementation_delay is int(3); comparing to literal 3.0 should be true.
        if err := EvaluateConstraint("implementation_delay == 3.0", vars); err != nil {
                t.Errorf("int==float comparison failed: %v", err)
        }
        // funding_level is int64(2_000_000).
        if err := EvaluateConstraint("funding_level >= 1500000", vars); err != nil {
                t.Errorf("int64 >= literal comparison failed: %v", err)
        }
        // interest_rate is float32(5.5).
        if err := EvaluateConstraint("interest_rate < 6", vars); err != nil {
                t.Errorf("float32 < literal comparison failed: %v", err)
        }
}

// TestValidateConstraints_RangeAndExpression verifies that ValidateConstraints
// surfaces both range violations and false Expression constraints, and that
// the error names the offending constraint ID.
func TestValidateConstraints_RangeAndExpression(t *testing.T) {
        min := 0.0
        max := 100.0
        t.Run("range_violation", func(t *testing.T) {
                s := Scenario{
                        Variables: []ScenarioVariable{
                                {Name: "tax_rate", Type: VariablePercentage, Value: float64(150), Minimum: &min, Maximum: &max},
                        },
                }
                err := ValidateConstraints(s)
                if err == nil {
                        t.Fatal("expected range violation; got nil")
                }
                if !strings.Contains(err.Error(), "above maximum") {
                        t.Errorf("expected 'above maximum' in error; got %v", err)
                }
        })
        t.Run("expression_false", func(t *testing.T) {
                s := Scenario{
                        Variables: []ScenarioVariable{
                                {Name: "tax_rate", Type: VariablePercentage, Value: float64(30), Minimum: &min, Maximum: &max},
                                {Name: "implementation_delay", Type: VariableInteger, Value: int(12)},
                        },
                        Constraints: []ScenarioConstraint{
                                {ID: "cn-delay-budget", Expression: "implementation_delay < 6"},
                        },
                }
                err := ValidateConstraints(s)
                if err == nil {
                        t.Fatal("expected constraint expression failure; got nil")
                }
                if !strings.Contains(err.Error(), "cn-delay-budget") {
                        t.Errorf("expected error to name the constraint ID; got %v", err)
                }
        })
        t.Run("expression_true", func(t *testing.T) {
                s := Scenario{
                        Variables: []ScenarioVariable{
                                {Name: "tax_rate", Type: VariablePercentage, Value: float64(30), Minimum: &min, Maximum: &max},
                                {Name: "implementation_delay", Type: VariableInteger, Value: int(3)},
                        },
                        Constraints: []ScenarioConstraint{
                                {ID: "cn-delay-budget", Expression: "tax_rate <= 50 && implementation_delay < 6"},
                        },
                }
                if err := ValidateConstraints(s); err != nil {
                        t.Fatalf("expected nil error; got %v", err)
                }
        })
}
