import sys
data = sys.stdin.read()
with open("/home/aos/temp/orchest-test-output.txt", "w+") as f:
    f.write(data)

