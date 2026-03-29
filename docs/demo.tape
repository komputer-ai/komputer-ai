# komputer.ai CLI Demo
# Run with: vhs docs/demo.tape

Output docs/demo.gif
Set Shell "bash"
Set FontSize 14
Set Width 900
Set Height 500
Set Padding 10
Set Theme "Catppuccin Mocha"
Set TypingSpeed 40ms
Set CursorBlink false

# Clean prompt for the recording (hidden)
Hide
Type `export PS1="$ "`
Enter
Sleep 2
Show
Type "clear"
Enter
Sleep 500ms

# 1. Start with list — show what we have
Type "komputer list"
Enter
Sleep 2s

# 2. Create an agent and stream output
Type "komputer run dev-agent 'Write a bash script that checks disk usage and warns if any partition is above 80%'"
Enter
Sleep 25s

# 3. List again — agent is there now
Type "komputer list"
Enter
Sleep 2s

# 4. Chat with the same agent — quick follow-ups
Type "komputer chat dev-agent"
Enter
Sleep 2s
Type "Make it also check memory usage"
Enter
Sleep 35s

# Exit chat
Ctrl+C
Sleep 1.5s

# 5. Spin up a manager that creates sub-agents (forms an office)
Type "komputer run research-team 'Research the top 3 Kubernetes security tools. Create a sub-agent for each one.'"
Enter
Sleep 45s

# 6. List offices — see the team
Type "komputer office list"
Enter
Sleep 2s

# 7. Get office details — members and costs
Type "komputer office get research-team"
Enter
Sleep 2s

# 8. Clean up
Type "komputer delete dev-agent"
Enter
Sleep 1.5s

Sleep 2s
