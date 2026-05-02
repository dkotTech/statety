package statety

import (
	"fmt"
	"html"
	"strings"
)

// DOT returns a Graphviz DOT representation of the state machine.
//
// Node appearance (self-documenting, no legend):
//   - regular state:     rounded box, blue fill
//   - final state:       green fill, "◆ final" row
//   - SaveOnEnter state: "↓ SaveOnEnter" row inside the node
//   - SaveOnExit state:  "↑ SaveOnExit"  row inside the node
func DOT[State comparable, Event comparable, Payload any](setup Setup[State, Event, Payload]) string {
	final := make(map[State]bool, len(setup.FinalStates))
	for _, s := range setup.FinalStates {
		final[s] = true
	}
	saveEnter := make(map[State]bool, len(setup.Config))
	saveExit := make(map[State]bool, len(setup.Config))
	for state, steps := range setup.Config {
		if steps.SaveOnEnter != nil {
			saveEnter[state] = true
		}
		if steps.SaveOnExit != nil {
			saveExit[state] = true
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

		name := html.EscapeString(fmt.Sprintf("%v", s))

		fillColor, borderColor := "#DDEEFF", "#336699"
		if final[s] {
			fillColor, borderColor = "#D6EAD6", "#2E7D32"
		}

		var lbl strings.Builder
		lbl.WriteString(`<TABLE BORDER="0" CELLBORDER="0" CELLSPACING="0" CELLPADDING="4">`)
		fmt.Fprintf(&lbl, `<TR><TD><B>%s</B></TD></TR>`, name)
		hasExtra := final[s] || saveEnter[s] || saveExit[s]
		if hasExtra {
			lbl.WriteString(`<HR/>`)
			if final[s] {
				lbl.WriteString(`<TR><TD ALIGN="LEFT"><FONT POINT-SIZE="10" COLOR="#2E7D32">&#9670; final</FONT></TD></TR>`)
			}
			if saveEnter[s] {
				lbl.WriteString(`<TR><TD ALIGN="LEFT"><FONT POINT-SIZE="10" COLOR="#444444">&#8595; SaveOnEnter</FONT></TD></TR>`)
			}
			if saveExit[s] {
				lbl.WriteString(`<TR><TD ALIGN="LEFT"><FONT POINT-SIZE="10" COLOR="#444444">&#8593; SaveOnExit</FONT></TD></TR>`)
			}
		}
		lbl.WriteString(`</TABLE>`)

		attrs := []string{
			fmt.Sprintf("label=<%s>", lbl.String()),
			"shape=box",
			`style="rounded,filled"`,
			fmt.Sprintf("fillcolor=%q", fillColor),
			fmt.Sprintf("color=%q", borderColor),
		}
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

	b.WriteString("}\n")
	return b.String()
}

func dotID[T comparable](v T) string {
	return fmt.Sprintf("s%x", fmt.Sprintf("%v", v))
}
