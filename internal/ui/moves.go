package ui

// waitForAnimations reports whether animations are blocking the game flow.
func (a *App) waitForAnimations() bool {
	if a.opts.NoAnimation {
		return false
	}
	return a.animations.isRunning()
}
