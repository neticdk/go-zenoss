load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = ["zenoss.go"],
    importpath = "github.com/neticdk/go-zenoss/zenoss",
    visibility = ["//visibility:public"],
    deps = ["@com_github_sirupsen_logrus//:go_default_library"],
)

go_test(
    name = "go_default_test",
    srcs = ["zenoss_test.go"],
    embed = [":go_default_library"],
)
