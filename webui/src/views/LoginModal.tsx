import { LockOutlined, UserOutlined } from "@ant-design/icons";
import { Button, Col, Form, Input, Modal, Row } from "antd";
import React, { useEffect, useState } from "react";
import { authenticationService, setAuthToken } from "../api";
import { LoginRequest } from "../../gen/ts/v1/authentication_pb";
import { useAlertApi } from "../components/Alerts";

export const LoginModal = () => {
  let defaultCreds = new LoginRequest();

  const [form] = Form.useForm();
  const alertApi = useAlertApi()!;

  useEffect(() => {
    authenticationService
      .login(
        new LoginRequest({
          username: "default",
          password: "password",
        }),
      )
      .then((loginResponse) => {
        alertApi.success(
          "No users configured yet, logged in with default credentials",
          5,
        );
        setAuthToken(loginResponse.token);
        setTimeout(() => {
          window.location.reload();
        }, 500);
      })
      .catch((e) => {});
  });

  const onFinish = async (values: any) => {
    const loginReq = new LoginRequest({
      username: values.username,
      password: values.password,
    });

    try {
      const loginResponse = await authenticationService.login(loginReq);
      setAuthToken(loginResponse.token);
      alertApi.success("Logged in", 5);
      setTimeout(() => {
        window.location.reload();
      }, 500);
    } catch (e: any) {
      alertApi.error("Login failed: " + (e.message ? e.message : "" + e), 10);
    }
  };

  return (
    <Modal
      open={true}
      width="40vw"
      title="Login"
      footer={null}
      closable={false}
    >
      <Form
        form={form}
        name="horizontal_login"
        layout="inline"
        onFinish={onFinish}
        style={{ width: "100%" }}
      >
        <Row justify="center" style={{ width: "100%" }}>
          <Col span={10}>
            <Form.Item
              name="username"
              rules={[
                { required: true, message: "Please input your username" },
              ]}
              style={{ width: "100%", paddingRight: "10px" }}
              initialValue={defaultCreds.username}
            >
              <Input
                prefix={<UserOutlined className="site-form-item-icon" />}
                placeholder="Username"
              />
            </Form.Item>
          </Col>

          <Col span={10}>
            <Form.Item
              name="password"
              rules={[
                { required: true, message: "Please input your password!" },
              ]}
              style={{ width: "100%", paddingRight: "10px" }}
              initialValue={defaultCreds.password}
            >
              <Input
                prefix={<LockOutlined className="site-form-item-icon" />}
                type="password"
                placeholder="Password"
              />
            </Form.Item>
          </Col>
          <Col span={4}>
            <Button type="primary" htmlType="submit" style={{ width: "100%" }}>
              Log in
            </Button>
          </Col>
        </Row>
      </Form>
    </Modal>
  );
};
