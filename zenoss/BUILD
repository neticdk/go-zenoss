load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

go_library(
    name = "go_default_library",
    srcs = [
        "types.go",
        "zenoss.go",
    ],
    importpath = "github.com/neticdk/go-zenoss/zenoss",
    visibility = ["//visibility:public"],
)

go_test(
    name = "go_default_test",
    srcs = ["zenoss_test.go"],
    embed = [":go_default_library"],
    deps = ["@com_github_stretchr_testify//assert:go_default_library"],
)
