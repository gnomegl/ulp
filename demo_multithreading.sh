#!/bin/bash

echo "=== ULP Multithreading Demo ==="
echo ""

# Create demo data
echo "Setting up demo data..."
cat > demo_creds.txt << 'EOF'
facebook.com:user1:pass123
google.com:admin:secret456
twitter.com:test:test789
instagram.com:demo:demo012
linkedin.com:john:john345
github.com:dev:dev678
reddit.com:mod:mod901
amazon.com:shop:shop234
netflix.com:watch:watch567
spotify.com:music:music890
discord.com:chat:chat123
slack.com:work:work456
zoom.com:meet:meet789
microsoft.com:office:office012
apple.com:mac:mac345
EOF

# Duplicate to create larger file
for i in {1..1000}; do
    cat demo_creds.txt >> large_demo.txt
done

echo "Created large_demo.txt with $(wc -l < large_demo.txt) lines"
echo ""

# Create directory structure
mkdir -p demo_dir
for i in {1..10}; do
    cp demo_creds.txt "demo_dir/creds_${i}.txt"
done

echo "=== 1. Clean Command with Workers ==="
echo "Running: ./ulp clean large_demo.txt cleaned.txt -w 4"
time ./ulp clean large_demo.txt cleaned.txt -w 4 2>&1 | head -5
echo ""

echo "=== 2. Dedupe Command with Workers ==="
echo "Running: ./ulp dedupe large_demo.txt deduped.txt -w 4 -d dupes.txt"
time ./ulp dedupe large_demo.txt deduped.txt -w 4 -d dupes.txt 2>&1 | head -10
echo ""

echo "=== 3. JSONL Command with Workers ==="
echo "Running: ./ulp jsonl demo_creds.txt -w 4"
./ulp jsonl demo_creds.txt -w 4 2>&1 | head -5
if [ -f demo_creds_ms_001.jsonl ]; then
    echo "Sample JSONL output:"
    head -1 demo_creds_ms_001.jsonl | jq '.' | head -10
fi
echo ""

echo "=== 4. CSV Command with Workers ==="
echo "Running: ./ulp csv demo_creds.txt -w 4 -o ."
./ulp csv demo_creds.txt -w 4 -o . 2>&1 | head -5
if [ -f demo_creds.csv ]; then
    echo "Sample CSV output:"
    head -3 demo_creds.csv
fi
echo ""

echo "=== 5. Directory Processing with Workers ==="
echo "Running: ./ulp clean demo_dir demo_output -w 4"
./ulp clean demo_dir demo_output -w 4 2>&1 | grep -E "Worker|Found|complete"
echo ""

echo "=== 6. Full Command with Workers ==="
echo "Running: ./ulp full demo_creds.txt --format jsonl -w 4"
./ulp full demo_creds.txt --format jsonl -w 4 2>&1 | head -5
echo ""

echo "=== Performance Comparison ==="
echo "Testing large file ($(wc -l < large_demo.txt) lines) with different worker counts:"
echo ""

echo -n "1 worker:  "
{ time ./ulp clean large_demo.txt out1.txt -w 1 2>/dev/null; } 2>&1 | grep real

echo -n "4 workers: "
{ time ./ulp clean large_demo.txt out4.txt -w 4 2>/dev/null; } 2>&1 | grep real

echo -n "8 workers: "
{ time ./ulp clean large_demo.txt out8.txt -w 8 2>/dev/null; } 2>&1 | grep real

echo ""
echo "=== Cleanup ==="
rm -rf demo_creds.txt large_demo.txt demo_dir demo_output cleaned.txt deduped.txt dupes.txt
rm -f demo_creds*.jsonl demo_creds*.csv out*.txt
echo "Demo complete!"