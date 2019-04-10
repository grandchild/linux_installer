# The final installer filename. You can override this when calling make like this:
#   make -DOUTPUT=Setup_ExampleApp_v9.2_ACME_64bit.bin
# or something similar.
OUTPUT=Setup
# The raw installer's filename (created by the parent linux-installer project)
INPUT=linux_installer

# The name of the folder where the data comes from
DATA_SRC_DIR = data
# The name of the folder containing the compressed data. If you change this you have to 
# change it in rice.go as well.
DATA_DIST_DIR = data_compressed

# Name of the "zip" command (might be different on some systems)
ZIP_EXE = zip

# Running just `make` depends on (i.e. creates) the OUTPUT file.
default: $(OUTPUT)

# Pre-compress the source data
$(DATA_DIST_DIR): $(DATA_SRC_DIR)
	mkdir -p $(DATA_DIST_DIR)
	cd $(DATA_SRC_DIR) ; $(ZIP_EXE) -r ../$(DATA_DIST_DIR)/data.zip .

# Make a copy of the raw installer & append the zipped data and resources to it.
$(OUTPUT): $(INPUT) clean $(DATA_DIST_DIR)
	cp $(INPUT) $(OUTPUT)
	./rice append --exec=$(OUTPUT)

# Remove previously created installer.
clean:
	rm -f $(OUTPUT)
