output "search_alb" {
  value = "${aws_elastic_beanstalk_environment.default.cname}"
}

output "mysql_master" {
  value = "${aws_rds_cluster_instance.master.endpoint}"
}

output "mysql_username" {
  value = "${random_id.username.b64}"
}

output "mysql_password" {
  value = "${random_id.password.b64}"
}
