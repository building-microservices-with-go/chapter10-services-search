# Grab the list of availability zones
data "aws_availability_zones" "available" {}

resource "random_id" "username" {
  keepers = {
    # Generate a new id each time we switch to a new cluster id
    id = "${aws_rds_cluster.default.id}"
  }

  byte_length = 8
}

resource "random_id" "password" {
  keepers = {
    # Generate a new id each time we switch to a new cluster id
    id = "${aws_rds_cluster.default.id}"
  }

  byte_length = 8
}

resource "aws_db_subnet_group" "default" {
  name       = "main"
  subnet_ids = ["${data.terraform_remote_state.main.vpc_subnets}"]
}

resource "aws_security_group" "mysql" {
  name        = "${var.namespace}-mysql"
  description = "Allow mysqsql inbound traffic"

  ingress {
    from_port   = 3306
    to_port     = 3306
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  vpc_id = "${data.terraform_remote_state.main.vpc_id}"
}

resource "aws_rds_cluster" "default" {
  cluster_identifier     = "aurora-cluster-demo"
  database_name          = "kittens"
  master_username        = "${random_id.username.b64}"
  master_password        = "${random_id.password.b64}"
  db_subnet_group_name   = "${aws_db_subnet_group.default.id}"
  vpc_security_group_ids = ["${aws_security_group.mysql.id}"]
  skip_final_snapshot    = "true"
}

resource "aws_rds_cluster_instance" "master" {
  count              = 1
  identifier         = "${var.namespace}-${var.application_name}-${count.index}"
  cluster_identifier = "${aws_rds_cluster.default.id}"
  instance_class     = "db.t2.small"

  db_subnet_group_name = "${aws_db_subnet_group.default.id}"
}

resource "null_resource" "cluster" {
  triggers {
    cluster_instance_ids = "${join(",", aws_rds_cluster_instance.master.*.id)}"
  }

  provisioner "file" {
    source      = "./templates/data.sql"
    destination = "~/search.sql"

    connection {
      type        = "ssh"
      host        = "${data.terraform_remote_state.main.ssh_host}"
      user        = "ubuntu"
      private_key = "${file("${var.private_key}")}"
    }
  }

  provisioner "remote-exec" {
    inline = [
      "mysql -h ${aws_rds_cluster_instance.master.endpoint} -u ${aws_rds_cluster.default.master_username} -p${aws_rds_cluster.default.master_password} < ~/search.sql",
    ]

    connection {
      type        = "ssh"
      host        = "${data.terraform_remote_state.main.ssh_host}"
      user        = "ubuntu"
      private_key = "${file("${var.private_key}")}"
    }
  }
}
