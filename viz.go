package statety

import (
	"fmt"
	"strings"
)

// DOT returns a Graphviz DOT representation of the state machine.
//
// Node legend:
//   - regular state:  rounded box, blue fill
//   - final state:    green fill, double border
//   - save state:     dashed border (route.Save != nil, combinable with final)
func DOT[State comparable, Event comparable, Payload any](setup Setup[State, Event, Payload]) string {
	final := make(map[State]bool, len(setup.FinalStates))
	for _, s := range setup.FinalStates {
		final[s] = true
	}
	save := make(map[State]bool, len(setup.Config))
	for state, steps := range setup.Config {
		if steps.Save != nil {
			save[state] = true
		}
	}

	var b strings.Builder

	b.WriteString("digraph statety {\n")
	b.WriteString("\trankdir=TB;\n")
	b.WriteString("\tforcelabels=true;\n")
	b.WriteString("\t__start [shape=point];\n")
	fmt.Fprintf(&b, "\t__start -> %s;\n", dotID(setup.StartState))

	written := map[State]bool{}
	writeNode := func(s State) {
		if written[s] {
			return
		}
		written[s] = true

		label := fmt.Sprintf("%v", s)
		attrs := []string{
			fmt.Sprintf("label=%q", label),
			"shape=box",
		}

		styles := []string{"rounded", "filled"}
		if save[s] {
			styles = append(styles, "dashed")
		}

		if final[s] {
			attrs = append(attrs,
				"peripheries=2",
				"fillcolor=\"#D6EAD6\"",
				"color=\"#2E7D32\"",
			)
		} else {
			attrs = append(attrs,
				"fillcolor=\"#DDEEFF\"",
				"color=\"#336699\"",
			)
		}

		attrs = append(attrs, fmt.Sprintf("style=%q", strings.Join(styles, ",")))
		fmt.Fprintf(&b, "\t%s [%s];\n", dotID(s), strings.Join(attrs, ", "))
	}

	writeNode(setup.StartState)
	for _, s := range setup.FinalStates {
		writeNode(s)
	}
	for state, steps := range setup.Config {
		writeNode(state)
		for event, next := range steps.Next {
			writeNode(next)
			fmt.Fprintf(&b, "\t%s -> %s [label=%q, decorate=true];\n",
				dotID(state), dotID(next), fmt.Sprintf("%v", event))
		}
	}

	b.WriteString(legend)
	b.WriteString("}\n")
	return b.String()
}

const legend = `
	subgraph cluster_legend {
		label="Legend";
		fontname="Helvetica";
		fontsize=12;
		style="rounded";
		color="#AAAAAA";
		margin=12;

		__l_regular [label="regular", shape=box, style="rounded,filled", fillcolor="#DDEEFF", color="#336699"];
		__l_final   [label="final",   shape=box, style="rounded,filled", fillcolor="#D6EAD6", color="#2E7D32", peripheries=2];
		__l_save    [label="save",    shape=box, style="rounded,filled,dashed", fillcolor="#DDEEFF", color="#336699"];

		__l_regular -> __l_final [style=invis];
		__l_final   -> __l_save  [style=invis];
	}
`

func dotID[T comparable](v T) string {
	return fmt.Sprintf("s%x", fmt.Sprintf("%v", v))
}
