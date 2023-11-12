import { Form, Modal, Input, Typography } from "antd";
import React, { useState } from "react";
import { useShowModal } from "../components/ModalManager";
import { Plan } from "../../gen/ts/v1/config.pb";

const nameRegex = /^[a-zA-Z0-9\-_ ]+$/;

export const AddPlanModal = ({
  template,
}: {
  template: Partial<Plan> | null;
}) => {
  const [confirmLoading, setConfirmLoading] = useState(false);
  const showModal = useShowModal();
  const [form] = Form.useForm();

  template = template || {};

  const handleOk = () => {
    setConfirmLoading(true);
    setTimeout(() => {
      showModal(null);
      setConfirmLoading(false);
    }, 2000);
  };

  const handleCancel = () => {
    showModal(null);
  };

  return (
    <>
      <Modal
        open={true}
        title="Add Plan"
        onOk={handleOk}
        confirmLoading={confirmLoading}
        onCancel={handleCancel}
      >
        <Form layout={"vertical"} autoComplete="off" form={form}>
          {/* Plan.id */}
          <Form.Item<Plan>
            hasFeedback
            name="id"
            label="Plan Name"
            initialValue={template.id}
            validateTrigger={["onChange", "onBlur"]}
            rules={[
              {
                required: true,
                message: "Please input plan name",
              },
              {
                pattern: nameRegex,
                message: "Invalid symbol",
              },
            ]}
          >
            <Input />
          </Form.Item>

          {/* Plan.repo */}
          <Form.Item<Plan>
            hasFeedback
            name="repo"
            label="Repo Name"
            initialValue={template.repo}
            validateTrigger={["onChange", "onBlur"]}
            rules={[
              {
                required: true,
                message: "Please input repo name",
              },
              {
                pattern: nameRegex,
                message: "Invalid symbol",
              },
            ]}
          >
            <Input />
          </Form.Item>

          {/* Plan.paths */}

          {/* Plan.excludes */}

          {/* Plan.cron */}

          {/* Plan.retention */}

          <Form.Item shouldUpdate label="Preview">
            {() => (
              <Typography>
                <pre>{JSON.stringify(form.getFieldsValue(), null, 2)}</pre>
              </Typography>
            )}
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
};
