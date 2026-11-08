import * as THREE from 'three'

// The rendered pose stays still while the marker moves within these limits.
const POSITION_TOLERANCE = 0.01
const ROTATION_TOLERANCE = THREE.MathUtils.degToRad(1.5)
const FOLLOW_TIME_SECONDS = 0.1

export class MarkerPoseStabilizer {
  readonly position = new THREE.Vector3()
  readonly quaternion = new THREE.Quaternion()
  private readonly samplePosition = new THREE.Vector3()
  private readonly sampleQuaternion = new THREE.Quaternion()
  private readonly sampleScale = new THREE.Vector3()
  private lastTime: number | null = null

  reset(): void {
    this.lastTime = null
  }

  update(matrix: THREE.Matrix4, time: number): void {
    matrix.decompose(this.samplePosition, this.sampleQuaternion, this.sampleScale)
    if (this.lastTime === null) {
      this.position.copy(this.samplePosition)
      this.quaternion.copy(this.sampleQuaternion)
      this.lastTime = time
      return
    }

    const elapsed = Math.min(0.1, Math.max(0, (time - this.lastTime) / 1000))
    this.lastTime = time
    const follow = 1 - Math.exp(-elapsed / FOLLOW_TIME_SECONDS)

    const distance = this.position.distanceTo(this.samplePosition)
    if (distance > POSITION_TOLERANCE) {
      this.position.lerp(this.samplePosition, (follow * (distance - POSITION_TOLERANCE)) / distance)
    }

    const angle = this.quaternion.angleTo(this.sampleQuaternion)
    if (angle > ROTATION_TOLERANCE) {
      this.quaternion.slerp(this.sampleQuaternion, (follow * (angle - ROTATION_TOLERANCE)) / angle)
    }
  }
}
