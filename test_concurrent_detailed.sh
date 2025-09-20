#!/bin/bash

echo "Creating test data..."

# Create a larger base file
cat > base_large.txt << 'EOF'
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
EOF

# Generate test file
> test_concurrent.txt
for i in {1..10000}; do
    sed "s/user/user${i}/g; s/pass/pass${i}/g" base_large.txt >> test_concurrent.txt
done

file_size=$(ls -lh test_concurrent.txt | awk '{print $5}')
line_count=$(wc -l < test_concurrent.txt)

echo "Test file: test_concurrent.txt"
echo "Size: $file_size"
echo "Lines: $line_count"
echo ""

echo "Running tests (showing stderr for worker messages)..."
echo ""

echo "=== Test 1: Sequential (1 worker) ==="
time ./ulp_concurrent clean test_concurrent.txt output_seq.txt -w 1

echo ""
echo "=== Test 2: Concurrent (4 workers) ==="
time ./ulp_concurrent clean test_concurrent.txt output_4w.txt -w 4

echo ""
echo "=== Test 3: Concurrent (8 workers) ==="
time ./ulp_concurrent clean test_concurrent.txt output_8w.txt -w 8

echo ""
echo "Verifying outputs are identical:"
if diff -q output_seq.txt output_4w.txt > /dev/null && diff -q output_4w.txt output_8w.txt > /dev/null; then
    echo "✓ All outputs are identical ($(wc -l < output_seq.txt) lines)"
else
    echo "✗ Outputs differ!"
    echo "Sequential: $(wc -l < output_seq.txt) lines"
    echo "4 workers:  $(wc -l < output_4w.txt) lines"
    echo "8 workers:  $(wc -l < output_8w.txt) lines"
fi

# Clean up
rm -f base_large.txt test_concurrent.txt output_*.txt

echo ""
echo "Test completed!"