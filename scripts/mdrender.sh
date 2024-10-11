#!/bin/bash

# You need the "glow" executable on your path to run this script

# Function to render markdown using glow
render_markdown() {
    local input="$1"
    glow --style auto <<< "$input"
}

input_buffer=""
line_count=0

# Save cursor position and hide it
tput sc
tput civis

# Clear screen once at the beginning
clear

# Read input line by line
while IFS= read -r line || [[ -n "$line" ]]; do
    # Increment line count
    ((line_count++))

    # Append the new line to input buffer
    input_buffer+="$line"$'\n'

    # Move cursor to home position (top-left corner)
    tput home

    # Render the current buffer
    render_markdown "$input_buffer"

    # Clear from cursor to end of screen
    tput ed

    # If we've rendered more than the terminal height, reset the screen
    if ((line_count % $(tput lines) == 0)); then
        clear
        tput home
    fi
done

# Show cursor again
tput cnorm
