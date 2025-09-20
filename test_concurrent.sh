#!/bin/bash

# Test script for concurrent ULP processing

echo "Creating test data..."
# Create a large test file with many credentials
cat > test_large.txt << 'EOF'
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
example.com:user1:pass1
site1.com:admin:password123
site2.org:user2:pass2
EOF

# Duplicate the content to make a larger file
for i in {1..100}; do
    cat test_large.txt >> test_input.txt
done

echo "Test file created with $(wc -l < test_input.txt) lines"
echo ""

# Test with different worker counts
echo "Testing with default workers (auto-detect CPU cores)..."
time ./ulp_concurrent clean test_input.txt test_output_default.txt
echo "Lines processed: $(wc -l < test_output_default.txt)"
echo ""

echo "Testing with 1 worker (sequential)..."
time ./ulp_concurrent clean test_input.txt test_output_1worker.txt -w 1
echo "Lines processed: $(wc -l < test_output_1worker.txt)"
echo ""

echo "Testing with 4 workers..."
time ./ulp_concurrent clean test_input.txt test_output_4workers.txt -w 4
echo "Lines processed: $(wc -l < test_output_4workers.txt)"
echo ""

echo "Testing with 8 workers..."
time ./ulp_concurrent clean test_input.txt test_output_8workers.txt -w 8
echo "Lines processed: $(wc -l < test_output_8workers.txt)"
echo ""

echo "Testing deduplication with concurrent processing..."
time ./ulp_concurrent dedupe test_input.txt test_output_dedupe.txt -w 4 -d test_duplicates.txt
echo "Unique lines: $(wc -l < test_output_dedupe.txt)"
echo "Duplicates found: $(wc -l < test_duplicates.txt 2>/dev/null || echo 0)"
echo ""

# Clean up
rm -f test_large.txt test_input.txt test_output_*.txt test_duplicates.txt

echo "Test completed!"