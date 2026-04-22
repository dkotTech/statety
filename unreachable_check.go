package statety

// UnreachableStates returns states declared in Config or FinalStates that
// cannot be reached from StartState via Next transitions. It is an optional
// lint — call it before NewMachine to surface dead states.
func UnreachableStates[State comparable, Event comparable, Payload any](setup Setup[State, Event, Payload]) []State {
	reached := map[State]struct{}{setup.StartState: {}}
	queue := []State{setup.StartState}
	for len(queue) > 0 {
		s := queue[0]
		queue = queue[1:]
		for _, next := range setup.Config[s].Next {
			if _, ok := reached[next]; ok {
				continue
			}
			reached[next] = struct{}{}
			queue = append(queue, next)
		}
	}

	seen := map[State]struct{}{}
	var unreachable []State
	check := func(s State) {
		if _, already := seen[s]; already {
			return
		}
		seen[s] = struct{}{}
		if _, ok := reached[s]; !ok {
			unreachable = append(unreachable, s)
		}
	}
	for s := range setup.Config {
		check(s)
	}
	for _, s := range setup.FinalStates {
		check(s)
	}
	return unreachable
}
