# Copyright 2013-2023, MACHBASE Corporation or its subsidiaries.
# All rights reserved.

include $(MACHBASEDEV_HOME)/test/itf/itf.mk

INCLUDES  += $(MACHBASE_HOME)/include $(NBASIS_HOME)/include $(MACHBASEDEV_HOME)/test/itf
LIBS      += jemalloc nbasistest machbaseuta machbasecli
LIBDIRS   += $(MACHBASE_HOME)/lib $(NBASIS_HOME)/lib$(COMPILE_BIT)
DEFINES   += NBP_CFG_DL_STATIC

WAPORT1  = $(shell echo $(ITF_PORT)+411 | bc)
WAPORT2  = $(shell echo $(ITF_PORT)+421 | bc)

EXECXX_TARGETS = append multi multi2 client server

append_NAME    = append
append_SRCS    = append.c

multi_NAME    = multi
multi_SRCS    = multi.c

multi2_NAME    = multi2
multi2_SRCS    = multi2.c


client_NAME    = client
client_SRCS    = client.c

server_NAME    = server
server_SRCS    = server.c

post_all:
	$(CP) $(BUILD_DIR)/append .
	$(CP) $(BUILD_DIR)/multi .
	$(CP) $(BUILD_DIR)/multi2 .
	$(CP) $(BUILD_DIR)/client .
	$(CP) $(BUILD_DIR)/server .

clean:
	@-rm -f append multi multi2 client server

include $(NBASISDEV_HOME)/core/makefiles/rule.mk
