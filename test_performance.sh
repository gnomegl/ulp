#!/bin/bash

# Performance test script for concurrent ULP processing with larger dataset

echo "Creating large test dataset..."

# Create a base file with various credential formats
cat > base_creds.txt << 'EOF'
example.com:user1:pass1
https://www.site1.com:admin:password123
site2.org|user2|pass2
www.site3.net:test:test123
https://site4.io:demo:demo456
android://app.example.com/:mobile:mobilepass
site5.com:user5:pass5
https://www.site6.com:admin6:password6
site7.org|user7|pass7
www.site8.net:test8:test8
https://site9.io:demo9:demo9
android://app2.example.com/:mobile2:mobilepass2
facebook.com:fbuser:fbpass123
google.com:googleuser:googlepass456
twitter.com:twitteruser:twitterpass789
instagram.com:instauser:instapass012
linkedin.com:linkeduser:linkedpass345
github.com:gituser:gitpass678
reddit.com:reddituser:redditpass901
amazon.com:amazonuser:amazonpass234
netflix.com:netflixuser:netflixpass567
spotify.com:spotifyuser:spotifypass890
discord.com:discorduser:discordpass123
slack.com:slackuser:slackpass456
zoom.com:zoomuser:zoompass789
microsoft.com:msuser:mspass012
apple.com:appleuser:applepass345
paypal.com:paypaluser:paypalpass678
ebay.com:ebayuser:ebaypass901
alibaba.com:aliuser:alipass234
tiktok.com:tiktokuser:tiktokpass567
snapchat.com:snapuser:snappass890
pinterest.com:pinuser:pinpass123
whatsapp.com:whatsuser:whatspass456
telegram.com:teleuser:telepass789
youtube.com:ytuser:ytpass012
twitch.tv:twitchuser:twitchpass345
dropbox.com:dropuser:droppass678
uber.com:uberuser:uberpass901
airbnb.com:airbnbuser:airbnbpass234
salesforce.com:salesuser:salespass567
oracle.com:oracleuser:oraclepass890
adobe.com:adobeuser:adobepass123
nvidia.com:nvidiauser:nvidiapass456
amd.com:amduser:amdpass789
intel.com:inteluser:intelpass012
ibm.com:ibmuser:ibmpass345
cisco.com:ciscouser:ciscopass678
dell.com:delluser:dellpass901
hp.com:hpuser:hppass234
EOF

# Generate a very large test file (100,000+ lines)
echo "Generating 100,000+ line test file..."
> large_test.txt
for i in {1..2000}; do
    # Add some variation to make it more realistic
    sed "s/user/user${i}/g; s/pass/pass${i}/g" base_creds.txt >> large_test.txt
done

total_lines=$(wc -l < large_test.txt)
echo "Created test file with $total_lines lines"
echo ""

# Create a test directory with multiple files
echo "Creating test directory with multiple files..."
mkdir -p test_dir
for i in {1..20}; do
    cp base_creds.txt "test_dir/creds_${i}.txt"
done
echo "Created 20 files in test_dir/"
echo ""

echo "=== PERFORMANCE COMPARISON ==="
echo ""

echo "1. Single file processing ($total_lines lines):"
echo "----------------------------------------"

echo "Testing with 1 worker (sequential)..."
time ./ulp_concurrent clean large_test.txt output_1w.txt -w 1 2>/dev/null
lines_1w=$(wc -l < output_1w.txt)

echo "Testing with 4 workers..."
time ./ulp_concurrent clean large_test.txt output_4w.txt -w 4 2>/dev/null
lines_4w=$(wc -l < output_4w.txt)

echo "Testing with 8 workers..."
time ./ulp_concurrent clean large_test.txt output_8w.txt -w 8 2>/dev/null
lines_8w=$(wc -l < output_8w.txt)

echo "Testing with auto workers (CPU cores)..."
time ./ulp_concurrent clean large_test.txt output_auto.txt 2>/dev/null
lines_auto=$(wc -l < output_auto.txt)

echo ""
echo "Results verification:"
echo "1 worker:    $lines_1w lines"
echo "4 workers:   $lines_4w lines"
echo "8 workers:   $lines_8w lines"
echo "Auto workers: $lines_auto lines"
echo ""

echo "2. Directory processing (20 files):"
echo "------------------------------------"

echo "Testing with 1 worker..."
time ./ulp_concurrent clean test_dir output_dir_1w -w 1 2>/dev/null

echo "Testing with 4 workers..."
time ./ulp_concurrent clean test_dir output_dir_4w -w 4 2>/dev/null

echo "Testing with 8 workers..."
time ./ulp_concurrent clean test_dir output_dir_8w -w 8 2>/dev/null

echo ""
echo "3. Deduplication test with large file:"
echo "---------------------------------------"

echo "Testing dedupe with 4 workers..."
time ./ulp_concurrent dedupe large_test.txt output_dedupe.txt -w 4 2>/dev/null
unique_lines=$(wc -l < output_dedupe.txt)
echo "Unique credentials found: $unique_lines"

echo ""
echo "4. JSONL conversion test:"
echo "-------------------------"

echo "Testing JSONL with 4 workers..."
time ./ulp_concurrent jsonl large_test.txt -w 4 2>/dev/null
if [ -f "large_test_ms_001.jsonl" ]; then
    jsonl_lines=$(wc -l < large_test_ms_001.jsonl)
    echo "JSONL lines created: $jsonl_lines"
fi

# Clean up
echo ""
echo "Cleaning up test files..."
rm -rf base_creds.txt large_test.txt test_dir output_* large_test_ms_*.jsonl

echo ""
echo "Performance test completed!"