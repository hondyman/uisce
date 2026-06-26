#!/bin/bash

# SemLayer Performance Profiling Script
# Captures CPU, memory, and mutex profiles for performance analysis

set -e

# Configuration
DURATION=${DURATION:-30s}
PPROF_PORT=${PPROF_PORT:-8081}
OUTPUT_DIR=${OUTPUT_DIR:-./profiles}
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# Create output directory
mkdir -p "$OUTPUT_DIR"

echo "🚀 Starting SemLayer Performance Profiling"
echo "Duration: $DURATION"
echo "Pprof Port: $PPROF_PORT"
echo "Output Directory: $OUTPUT_DIR"
echo "Timestamp: $TIMESTAMP"
echo

# Function to capture profile
capture_profile() {
    local profile_type=$1
    local filename="$OUTPUT_DIR/${profile_type}_profile_$TIMESTAMP.pprof"

    echo "📊 Capturing $profile_type profile..."

    case $profile_type in
        "cpu")
            curl -s "http://localhost:$PPROF_PORT/debug/pprof/profile?seconds=$(echo $DURATION | sed 's/s//')" > "$filename"
            ;;
        "heap")
            curl -s "http://localhost:$PPROF_PORT/debug/pprof/heap" > "$filename"
            ;;
        "mutex")
            curl -s "http://localhost:$PPROF_PORT/debug/pprof/mutex" > "$filename"
            ;;
        "block")
            curl -s "http://localhost:$PPROF_PORT/debug/pprof/block" > "$filename"
            ;;
        "goroutine")
            curl -s "http://localhost:$PPROF_PORT/debug/pprof/goroutine" > "$filename"
            ;;
    esac

    if [ -f "$filename" ] && [ -s "$filename" ]; then
        echo "✅ $profile_type profile saved to $filename"
        echo "   Size: $(du -h "$filename" | cut -f1)"
    else
        echo "❌ Failed to capture $profile_type profile"
    fi
}

# Function to analyze profile
analyze_profile() {
    local profile_type=$1
    local filename="$OUTPUT_DIR/${profile_type}_profile_$TIMESTAMP.pprof"

    if [ ! -f "$filename" ]; then
        echo "⚠️  Profile file $filename not found"
        return
    fi

    echo "🔍 Analyzing $profile_type profile..."

    case $profile_type in
        "cpu")
            go tool pprof -top "$filename" | head -20
            ;;
        "heap")
            go tool pprof -top "$filename" | head -20
            ;;
        "mutex")
            go tool pprof -top "$filename" | head -20
            ;;
    esac
}

# Check if pprof server is running
echo "🔍 Checking if pprof server is accessible..."
if ! curl -s "http://localhost:$PPROF_PORT/debug/pprof/" > /dev/null; then
    echo "❌ Pprof server not accessible at http://localhost:$PPROF_PORT"
    echo "   Make sure the SemLayer service is running with pprof enabled"
    echo "   Use: ./semlayer -pprof-port=$PPROF_PORT"
    exit 1
fi

echo "✅ Pprof server is accessible"
echo

# Capture profiles
echo "📈 Starting profile capture..."
capture_profile "cpu"
capture_profile "heap"
capture_profile "mutex"
capture_profile "block"
capture_profile "goroutine"

echo
echo "📋 Profile Summary:"
echo "=================="
ls -la "$OUTPUT_DIR"/*"$TIMESTAMP".pprof

echo
echo "🔬 Analyzing top CPU consumers..."
analyze_profile "cpu"

echo
echo "💾 Analyzing memory usage..."
analyze_profile "heap"

echo
echo "🔒 Analyzing mutex contention..."
analyze_profile "mutex"

echo
echo "📊 To analyze profiles interactively:"
echo "   go tool pprof $OUTPUT_DIR/cpu_profile_$TIMESTAMP.pprof"
echo "   go tool pprof $OUTPUT_DIR/heap_profile_$TIMESTAMP.pprof"
echo "   go tool pprof $OUTPUT_DIR/mutex_profile_$TIMESTAMP.pprof"
echo
echo "🌐 View profiles in browser:"
echo "   go tool pprof -http=:8080 $OUTPUT_DIR/cpu_profile_$TIMESTAMP.pprof"
echo
echo "✅ Profiling complete! Profiles saved to $OUTPUT_DIR"
